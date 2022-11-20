const path = require("path");
module.exports = {
  mode: "production",
  target: "es2019",
  devtool: "cheap-module-source-map",
  optimization: {
    sideEffects: true,
  },
  resolve: {
    extensions: [".js"],
  },
  output: {
    libraryTarget: "umd",
    globalObject: "this",
    filename: "index.js",
    path: path.join(__dirname, "build"),
    library: "Suborbital",
    chunkFormat: "array-push",
  },
};
