var webpack = require('webpack');

/*
 * SPRUCE_ENV is defined on the command line. For example, the following commands will build for dev and prod, respectively:
 * 
 * SPRUCE_ENV='dev' ./node_modules/.bin/webpack -d
 * SPRUCE_ENV='prod' ./node_modules/.bin/webpack -p
 */
isDev = process.env.SPRUCE_ENV == "dev"

module.exports = {
	context: __dirname + "/src",
	
	entry: {
		DemoRequestForm: "./DemoRequestForm.js",
		WhitepaperForm: "./WhitepaperForm.js"
	},

	output: {
		filename: isDev ? "[name].dev.js" : "[name].min.js",
		path: __dirname + "../../../static/js",
	},

	module: {
		loaders: [{
			test: /\.jsx?$/, // A regexp to test the require path. accepts either js or jsx
			loader: 'babel' // The module to load. "babel" is short for "babel-loader"
		}]
	},

	plugins: [
		new webpack.ProvidePlugin({
			'es6-promise': 'es6-promise',
			'fetch': 'imports?this=>global!exports?global.fetch!whatwg-fetch'
		}),
		new webpack.optimize.UglifyJsPlugin({
			compress: {
				warnings: false
			}
		})
	]
}