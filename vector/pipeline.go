package vector

import (
	"context"
	"golang.org/x/sync/errgroup"
)

// Each worker get its own copy of the pipeline.
type pipeline struct {
	pipes []pipe
	sink  sink
}

type source interface {
	// Returns io.EOF when done.
	pull() (*Vector, error)
}

type pipeStatus int

const (
	needInput pipeStatus = iota
	haveOutput
	done
)

// A pipe takes an input vector and returns an output vector.
// But we want to make efficient use of cache so:
// * If the output vector would be too small, buffer it until the next input.
// * If the output vector would be too large, return part of the output and wait to be called again.
type pipe interface {
	// If returns `needInput`, then we should call `work` again with a non-nil vector.
	// If returns `haveOutput`, then we should call `takeOutput` before calling `work` again with a nil vector.
	// If returns `done`, then we should never call `work` or `takeOutput` again.
	work(input *Vector) (pipeStatus, error)
	// Take the pipes output buffer, replacing it with an empty buffer.
	takeOutput() *Vector
}

type sink interface {
	// Input vector must not be nil.
	push(input *Vector) error
	finish() error
}

// Run each pipeline in a worker until either:
// * All output has been produced and finish has been called on all sinks.
// * An error is returned.
func runPipelines(ctx context.Context, source source, pipelines []pipeline) error {
	group, ctx := errgroup.WithContext(ctx)
	sourceChans := make([]chan *Vector, len(pipelines))
	for i := range pipelines {
		sourceChans[i] = make(chan *Vector, 2)
		group.Go(func() error { return runPipeline(ctx, &pipelines[i], sourceChans[i]) })
	}
	group.Go(func() error { return runSource(ctx, source, sourceChans) })
	return group.Wait()
}

// Pull vectors from `source` and feed them to workers.
func runSource(ctx context.Context, source source, sourceChans []chan *Vector) error {
	// TODO Ideally we want the workers to use work-stealing queues, but round-robin is fine for now.
	var next int
	for {
		if ctx.Err() != nil {
			return nil
		}
		input, err := source.pull()
		if err != nil {
			return err
		}
		sourceChans[next] <- input
		next = (next + 1) % len(sourceChans)
	}
}

// Run `pipeline` until either:
// * All output has been produced and finish has been called on the sink.
// * An error is returned.
func runPipeline(ctx context.Context, pipeline *pipeline, sourceChan chan *Vector) error {
	if ctx.Err() != nil {
		return nil
	}

	// Push all input.
	for input := range sourceChan {
		// TODO Vector size for pipes should be much smaller than size for work-stealing. Break into chunks here.
		done, err := runPipe(ctx, pipeline, 0, input)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	// Flush any remaining output.
	runPipe(ctx, pipeline, 0, nil)

	return pipeline.sink.finish()
}

func runPipe(ctx context.Context, pipeline *pipeline, pipeIndex int, input *Vector) (bool, error) {
	if ctx.Err() != nil {
		return false, nil
	}

	// If input came from last pipe, push it to the sink.
	if pipeIndex == len(pipeline.pipes) {
		var err error
		if input != nil {
			err = pipeline.sink.push(input)
		}
		return false, err
	}

	pipe := pipeline.pipes[pipeIndex]

	// If no more input is coming then just flush remaining output.
	if input == nil {
		output := pipe.takeOutput()
		if output != nil {
			done, err := runPipe(ctx, pipeline, pipeIndex+1, output)
			if err != nil {
				return false, err
			}
			if done {
				return true, nil
			}
		}
		return runPipe(ctx, pipeline, pipeIndex+1, nil)
	}

	// Push all outputs for this one input.
	for {
		status, err := pipe.work(input)
		if err != nil {
			return false, err
		}
		switch status {
		case needInput:
			return false, nil
		case haveOutput:
			output := pipe.takeOutput()
			done, err := runPipe(ctx, pipeline, pipeIndex+1, output)
			if err != nil {
				return false, err
			}
			if done {
				return true, nil
			}
			// Call pipe.work again in the next loop iteration with a nil input.
			input = nil
		case done:
			return true, nil
		default:
			panic("Unreachable")
		}
	}
}
