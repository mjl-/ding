// don't warn about "use strict"
/* jshint -W097 */
/* global app, api, _ */
'use strict';

app.controller('Repo', function($scope, $rootScope, $q, $location, $timeout, Msg, Util, repo, builds) {
	$rootScope.breadcrumbs = Util.crumbs([
		Util.crumb('repo/' + repo.name, 'Repo ' + repo.name)
	]);

	$scope.repo = repo;
	$scope.repoUID = repo.uid !== null;
	$scope.builds = builds;
	$scope.releaseBuilds = _.filter($scope.builds, function(b) { return b.released; });

	function updateReleaseBuilds() {
		$scope.releaseBuilds = _.filter($scope.builds, function(b) { return b.released; });
	}
	updateReleaseBuilds();

	$scope.$on('removeRepo', function(x, e) {
		if (e.repo_name === $scope.repo.name) {
			$location.path('/');
		}
	});

	$scope.$on('build', function(x, e) {
		var b = e.build;
		var repoName = e.repo_name;
		if (repoName !== $scope.repo.name) {
			return;
		}
		$timeout(function() {
			for (var i = 0; i < $scope.builds.length; i++) {
				var bb = $scope.builds[i];
				if (bb.id === b.id) {
					$scope.builds[i] = b;
					updateReleaseBuilds();
					return;
				}
			}
			$scope.builds.unshift(b);
			updateReleaseBuilds();
		});
	});

	$scope.$on('removeBuild', function(x, e) {
		var build_id = e.build_id;
		$timeout(function() {
			$scope.builds = _.filter($scope.builds, function(b) {
				return b.id !== build_id;
			});
		});
	});


	$scope.removeRepo = function() {
		return Msg.confirm('Are you sure?', function() {
			return api.removeRepo($rootScope.password(), repo.name)
			.then(function() {
				$location.path('/');
			});
		});
	};

	$scope.clearRepoHomedir = function() {
		return Msg.confirm('Are you sure?', function() {
			return api.clearRepoHomedir($rootScope.password(), repo.name);
		});
	};

	$scope.save = function() {
		var repo = _.clone($scope.repo);
		repo.uid = $scope.repoUID ? 1 : null;
		return api.saveRepo($rootScope.password(), repo)
		.then(function(r) {
			$scope.repo = r;
		});
	};

	$scope.removeBuild = function(build) {
		return Msg.confirm('Are you sure?', function() {
			return api.removeBuild($rootScope.password(), build.id)
			.then(function() {
				$scope.builds = _.filter($scope.builds, function(b) {
					return b.id !== build.id;
				});
			});
		});
	};

	$scope.createBuild = function(repoName, branch, commit) {
		return api.createBuild($rootScope.password(), repoName, branch, commit)
		.then(function(nbuild) {
			$location.path('/repo/' + repoName + '/build/' + nbuild.id + '/');
		});
	};

	$scope.createBuildLowPrio = function(repoName, branch, commit) {
		return api.createBuildLowPrio($rootScope.password(), repoName, branch, commit);
	};

	$scope.cleanupBuilddir = function(build) {
		return api.cleanupBuilddir($rootScope.password(), repo.name, build.id)
		.then(function(nbuild) {
			$scope.builds = _.map($scope.builds, function(b) {
				return b.id === build.id ? nbuild : b;
			});
		});
	};
});
