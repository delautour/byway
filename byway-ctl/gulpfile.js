const gulp = require('gulp')
const gutil = require("gulp-util");
const webpack = require("webpack");
const child = require('child_process');
const notifier = require('node-notifier');
const WebpackDevServer = require("webpack-dev-server");

function runProc(name, args) {
    var proc
    return () => {
        if (proc) {
            proc.kill()
        }

        /* Spawn proc */
        proc = child.spawn(name, args);

        /* Pretty print server log output */
        proc.stdout.on('data', (data) => {
            var lines = data.toString().split('\n')
            for (var l in lines)
                if (lines[l].length)
                    gutil.log(lines[l]);
        });

        /* Print errors to stdout */
        proc.stderr.on('data', (data) => {
            process.stdout.write(data.toString());
        });
    }
}

gulp.task('webpack-dev-server', () => {
    // Start a webpack-dev-server
    const config = require('./webpack.config')

    config.plugins.push(new webpack.HotModuleReplacementPlugin());

    config.entry = [config.entry,
         require.resolve("webpack-dev-server/client/") + '?http://localhost:8080',
        'webpack/hot/dev-server'
    ]
    const compiler = webpack(config);

    new WebpackDevServer(compiler, {
        contentBase: 'dev-server',
        historyApiFallback: true,
        hot: true,
        inline: true,
        stats: false,
        quite: true,
        // server and middleware options
    }).listen(8080, "localhost", (err) => {
        if (err) throw new gutil.PluginError("webpack-dev-server", err);
        // Server listening
        gutil.log("[webpack-dev-server]");

        // keep the server alive or continue?
        // callback();
    });
})

gulp.task('ctl:build', () => {
    child.spawnSync('go', ['install'])
})
gulp.task('ctl:run', ['ctl:build'], runProc("byway-ctl"))

gulp.task('proxy:build', () => {
    child.spawnSync('go', ['install', '../byway-proxy'])
})
gulp.task('proxy:run', ['proxy:build'], runProc('byway-proxy'))

gulp.task('dev', ['ctl:run', 'proxy:run', 'webpack-dev-server'], () => {

    gulp.watch([
        '../**/*.go'
    ], ['ctl:run'])

    gulp.watch([
        '../**/*.go'
    ], ['proxy:run'])

    


})

gulp.task('default', () => {

})