var webpack = require('webpack');

module.exports = {
	context: __dirname + "/src/js",
	entry: { ApplyForm: "./ApplyForm" },

	devtool: 'source-map',

	output: {
		filename: "[name].js",
		path: __dirname + "/build/js",
	},

	module: {
		loaders: [{
			test: /\.jsx?$/, // A regexp to test the require path. accepts either js or jsx
			loader: 'babel' // The module to load. "babel" is short for "babel-loader"
		}]
	},

	plugins: [
		new webpack.ProvidePlugin({
			'fetch': 'imports?this=>global!exports?global.fetch!whatwg-fetch'
		})
	]
}