package winexec

import (
	"fmt"
	"os"
	"os/exec"
	"unsafe"

	"github.com/alexbrainman/ps"
	"github.com/alexbrainman/ps/winapi"
)

// ensureSpawnedProcessTermination ensures that when this Go process terminates, the
// launched process will also be terminated. It does so via the Windows
// job objects api:
// https://docs.microsoft.com/en-us/windows/win32/procthread/job-objects
//
// See this Go issue for discussion about the challenge of process management
// on Windows, and the mention of the ps & winapi package used here:
// https://github.com/golang/go/issues/17608
func ensureSpawnedProcessTermination() error {
	// Create an unnamed job object; no name is necessary since no other
	// process needs to find or interact with it.
	jo, err := ps.CreateJobObject("")
	if err != nil {
		return err
	}

	// We add ourselves so that any process we launch will automatically be
	// added to the job.
	err = jo.AddCurrentProcess()
	if err != nil {
		return err
	}

	// We set the "kill on job close" option for the job, so that when the
	// last handle to the job is closed, all of the processes in the job
	// will be terminated.
	// This process is the only one with a handle to the job object, and we
	// intentionally leave it open. Like other handles, it will be closed
	// automatically when this process terminates. When that occurs, the
	// 'kill on job close' option will trigger the termination of any
	// spawned processes.
	limitInfo := winapi.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: winapi.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: winapi.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	return winapi.SetInformationJobObject(jo.Handle, winapi.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&limitInfo)), uint32(unsafe.Sizeof(limitInfo)))
}

func winexec(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("expected command to execute")
	}

	err := ensureSpawnedProcessTermination()
	if err != nil {
		return err
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
