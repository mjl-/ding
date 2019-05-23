/* jshint -W097 */ // for "use strict"
/* global app, window, console */
'use strict';

app
.directive('buildtime', function($timeout) {
	return {
		restrict: 'E',
		template: '<span><span ng-if="start">{{ elapsed.toFixed(1) }}s</span><span ng-if="!finish">...</span></span>',
		scope: {
			'start': '=',
			'finish': '='
		},
		link: function(scope, element) {
			var timeoutID;
			var scheduleUpdate = function() {
				var spentMS = new Date().getTime() - new Date(scope.start).getTime();
				var wait = spentMS < 5*1000 ? 100 : 1000;
				timeoutID = window.setTimeout(function() {
					timeoutID = null; // Cause another schedule by render, if needed.
					$timeout(render);
				}, wait);
			};

			var render = function() {
				var finish = scope.finish ? new Date(scope.finish) : new Date();
				scope.elapsed = (finish.getTime() - new Date(scope.start).getTime()) / 1000;
				if (scope.start && !scope.finish && !timeoutID) {
					scheduleUpdate();
				}
			};

			scope.$watch(function() {
				return [scope.start, scope.finish];
			}, render, true);

			scope.$on('$destroy', function() {
				if (timeoutID) {
					window.clearTimeout(timeoutID);
					timeoutID = null;
				}
			});
		}
	};
});
