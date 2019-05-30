/* jshint -W097 */ // for "use strict"
/* global app, window, console */
'use strict';

app
.directive('age', function($timeout) {
	return {
		restrict: 'E',
		template: '{{ age }}',
		scope: {
			'time': '='
		},
		link: function(scope, element) {
			var timeoutID;

			scope.$on('$destroy', function() {
				if (timeoutID) {
					window.clearTimeout(timeoutID);
					timeoutID = null;
				}
			});

			var update = function() {
				var sec = parseInt(new Date().getTime() - new Date(scope.time).getTime()) / 1000;
				var age;
				var wait = 1 * 1000;
				if (sec < 60) {
					age = 'now';
				} else if (sec < 120*60) {
					age = Math.round(sec / 60) + 'm';
				} else if (sec < 48*3600) {
					age = Math.round(sec / 3600) + 'h';
					wait = 60 * 1000;
				} else if (sec < 21*24*3600) {
					age = Math.round(sec / (24*3600)) + 'd';
					wait = 3600 * 1000;
				} else {
					age = Math.round(sec / (7*24*3600)) + 'w';
					wait = 24 * 3600 * 1000;
				}
				$timeout(function() {
					scope.age = age;
				});
				timeoutID = window.setTimeout(update, wait);
			};

			update();
		}
	};
});
