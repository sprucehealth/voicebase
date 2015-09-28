## Getting Started

`npm install -g webpack` to be able to call `webpack` from the command line
`npm install -g gulp` to be able to call `gulp` from the command line

`npm install` to install the rest of the dependencies

## Development

### Go web server

Build and runs the web server:

```
go build
./curbside --env="dev" --listen_port="8100"
```

The assets will be served locally. See below for additional flags/options.

If you have [`justrun`](https://github.com/jmhodges/justrun) installed, the following will watch for updates in the top-level directory, as well as the `templates` directory. Once a change is made, it will stop, rebuild, and restart the server.

justrun -c 'go build && ./derm-practice-extension' -i derm-practice-extension * ./templates/*

### JavaScript (built with `webpack`)

`webpack --watch` to rebuild upon changes

#### More Webpack Commands

* `webpack` for building once for development
* `-p` for building once for production (minification)
* `-d` to include source maps (aka "debug symbols")
* * `--colors` for making things pretty
* `--watch` for continuous incremental build in development (fast!) -- by default, includes source maps and color

### CSS / SASS (built with `gulp`)

`gulp watch` listens for changes to CSS and automatically reruns `gulp styles` automatically

#### More Gulp Commands

* `gulp styles` builds and minifies the SASS
* `gulp clean` deletes build products from `gulp styles`
* `gulp` to run the above tasks (`clean`, then `styles`)

### Images and Fonts

* `gulp` will copy `src/img` and `src/fonts` to their respective folders within `build`

## Production

To build:

`npm run build`
`go build`

To run:

`./curbside-website --env="prod" --listen_port="8100" --static_resource_url="https://dlzz6qy5jmbag.cloudfront.net/curbside/{BuildNumber}" --slack_webhook_url="INSERT_WEBHOOK_URL_HERE"`

where `{BuildNumber}` gets replaced at run-time with the actual build number.