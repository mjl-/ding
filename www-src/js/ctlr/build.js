// don't warn about "use strict"
/* jshint -W097 */
/* global app, api, _, console, window, document, $ */
'use strict';

app.controller('Build', function($scope, $rootScope, $q, $location, $timeout, Msg, Util, repo, buildResult) {
	$rootScope.breadcrumbs = Util.crumbs([
		Util.crumb('repo/' + repo.name, 'Repo ' + repo.name),
		Util.crumb('build/' + buildResult.build.id + '/', 'Build ' + buildResult.build.id)
	]);

	$scope.repo = repo;
	$scope.buildResult = buildResult;
	$scope.build = buildResult.build;
	$scope.steps = buildResult.steps;

	$scope.newerBuild = null;
	$scope.hideNewerBuild = function() {
		$scope.hideBuildID = $scope.newerBuild.id;
	};

	$scope.$on('build', function(x, e) {
		var b = e.build;
		if (b.id !== $scope.build.id) {
			if (e.repo_name === repo.name && (!$scope.newerBuild || (e.build.start && e.build.id > $scope.newerBuild.id))) {
				$timeout(function() {
					$scope.newerBuild = e.build;
				});
			}
			return;
		}
		$timeout(function() {
			$scope.build = b;
		});
	});

	$scope.$on('removeBuild', function(x, e) {
		if (e.build_id === $scope.build.id) {
			$location.path('/repo/' + repo.name + '/');
			return;
		}
	});

	$scope.$on('output', function(x, e) {
		if (e.build_id !== $scope.build.id) {
			return;
		}
		$timeout(function() {
			var step = _.find($scope.steps, {name: e.step});
			if (!step) {
				step = {
					name: e.step,
					output: '',
					// nsec: 0,
					_start: new Date().getTime()
				};
				$scope.steps.push(step);
			}
			var slack = 3;
			var scroll = $(window).scrollTop() + $(window).height()  >= $(document).height() - slack;
			step.output += e.text;
			if (scroll) {
				$timeout(function() {
					$(window).scrollTop($(document).height() - $(window).height());
				});
			}
		});
	});


	$scope.removeBuild = function() {
		var build = $scope.build;
		return Msg.confirm('Are you sure?', function() {
			return api.removeBuild($rootScope.password(), build.id)
			.then(function() {
				$location.path('/repo/' + repo.name + '/');
			});
		});
	};

	$scope.retryBuild = function() {
		var build = $scope.build;
		return api.createBuild($rootScope.password(), repo.name, build.branch, build.commit_hash)
		.then(function(nbuild) {
			$location.path('/repo/' + repo.name + '/build/' + nbuild.id + '/');
		});
	};

	$scope.release = function() {
		var build = $scope.build;
		return api.createRelease($rootScope.password(), repo.name, build.id)
		.then(function(nbuild) {
			$location.path('/repo/' + repo.name + '/release/' + build.id + '/');
		});
	};

	$scope.cleanupBuilddir = function() {
		var build = $scope.build;
		return api.cleanupBuilddir($rootScope.password(), repo.name, build.id)
		.then(function(nbuild) {
			$location.path('/repo/' + repo.name + '/');
		});
	};

	$scope.cancelBuild = function() {
		return api.cancelBuild($rootScope.password(), repo.name, $scope.build.id);
	};
});
