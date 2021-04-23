// Mimic the build recipe in Makefile. Written in node so it works on
// Windows and *nix when trying to npm install zq.
const child_process = require("child_process")
const fs = require("fs")

const getVersion = () => {
  const GIT_COMMAND = "git describe --tags --dirty --always"
  let cmdOut = "prepack-unknown"
  try {
    cmdOut = child_process.execSync(GIT_COMMAND, { stdio: "pipe" })
  } catch (e) {
    console.log(`unable to run "${GIT_COMMAND}": ${e.toString().trim()}`)
  }
  return cmdOut.toString().trim()
}

const getLdflags = (version) => `-s -X github.com/brimdata/zed/cli.Version=${version}`

const getBuildCommand = (options) =>
  // Double-quotes work both in Windows and *nix shells
  `go build -ldflags="${options.ldflags}" -o dist ./cmd/...`

const mkdir_p = (path) => {
  if (!fs.existsSync(path)) {
    fs.mkdirSync(path)
  }
}

const build = () => {
  let version = getVersion()
  let ldflags = getLdflags(version)
  let buildCommand = getBuildCommand({ ldflags: ldflags })
  mkdir_p("dist")
  console.log(buildCommand)
  // Just let any failure here bubble up.
  child_process.execSync(buildCommand)
}

build()
