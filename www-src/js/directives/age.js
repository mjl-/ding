/* jshint -W097 */ // for "use strict"
/* global app, console */
'use strict';

app
.directive('age', function() {
	return {
		restrict: 'E',
		template: '{{ age }}',
		scope: {
			'time': '='
		},
		link: function(scope, element) {
			var sec = parseInt(new Date().getTime() - new Date(scope.time).getTime()) / 1000;
			var age;
			if (sec < 60) {
				age = Math.round(sec) + 's';
			} else if (sec < 120*60) {
				age = Math.round(sec / 60) + 'm';
			} else if (sec < 48*3600) {
				age = Math.round(sec / 3600) + 'h';
			} else if (sec < 21*24*3600) {
				age = Math.round(sec / (24*3600)) + 'd';
			} else {
				age = Math.round(sec / (7*24*3600)) + 'w';
			}
			scope.age = age;
		}
	};
});
