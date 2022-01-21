const path = require("path");
const os = require("os");

function getPath(name) {
  if (os.platform() === "win32") name += ".exe";
  return path.join(__dirname, "..", "dist", name);
}

module.exports = {
  getPath,
};
