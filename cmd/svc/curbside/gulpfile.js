// General
var gulp = require('gulp'),
	rename = require('gulp-rename'),
	notify = require('gulp-notify'),
	del = require('del'),
	sourcemaps = require('gulp-sourcemaps');

// Sass requirements
var sass = require('gulp-ruby-sass'),
	autoprefixer = require('gulp-autoprefixer'),
	minifycss = require('gulp-minify-css');

gulp.task('styles', function() {
	return sass('src/css', {
		style: 'expanded',
		sourcemap: true
	})
	.pipe(sourcemaps.init())
	.pipe(autoprefixer())
	.pipe(rename({suffix: '.min'}))
	.pipe(minifycss())
	.pipe(sourcemaps.write())
	.pipe(gulp.dest('build/css'));
});

gulp.task('fonts', function() {
	return gulp.src('../../../resources/static/fonts/*')
		.pipe(gulp.dest('build/fonts'));
});

gulp.task('clean', function(cb) {
	del([
		'build/fonts',
		'build/css',
		'build/img',
		'build/templates',
	], cb);
});

gulp.task('copy', function () {
    return gulp.src(['src/img/**/*', 'src/templates/**/*'], {
        base: 'src'
    }).pipe(gulp.dest('build'));
});

gulp.task('default', ['clean'], function() {
	gulp.start('styles');
	gulp.start('fonts');
	gulp.start('copy');
});

gulp.task('watch', function() {

	// Run tasks once immediately
	gulp.start('default');

	// Watch for changes
	gulp.watch('src/css/**/*.scss', ['styles']);
});