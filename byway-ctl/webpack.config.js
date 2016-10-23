const path = require('path')
module.exports = {
    entry: path.resolve('./src/main'),
    output: {
        path: path.resolve('./dist')
    },
    resolve: {
        // Add `.ts` and `.tsx` as a resolvable extension.
        extensions: ['', '.webpack.js', '.web.js', '.ts', '.tsx', '.js']
    },
    plugins: [],
    module: {
        loaders: [
            { test: /\.tsx?$/, loaders: ['react-hot-loader/webpack', 'ts'] },
            { test: /\.scss$/, loaders: ['style', 'css?modules', 'sass'] }
        ]
    },
    devServer: {
        hot: true,
        inline: true,
    }

}