// don't warn about "use strict"
/* jshint -W097 */
/* global app, api, _, console */
'use strict';

app.controller('Index', function($scope, $rootScope, $q, $uibModal, $location, $timeout, Msg, Util, repoBuilds) {
	$rootScope.breadcrumbs = Util.crumbs([]);

	$scope.repoBuilds = repoBuilds;

	$scope.$on('repo', function(x, e) {
		$timeout(function() {
			var r = e.repo;
			var rr = _.find($scope.repoBuilds, function(rb) {
				return rb.repo.name === r.name;
			});
			if (rr) {
				rr.repo = r;
				return;
			}
			$scope.repoBuilds.push({
				repo: r,
				builds: []
			});
		});
	});

	$scope.$on('removeRepo', function(x, e) {
		$timeout(function() {
			$scope.repoBuilds = _.filter($scope.repoBuilds, function(rb) {
				return rb.repo.name !== e.repo_name;
			});
		});
	});

	function buildsCleanup(rb) {
		// For each branch, we want to newest finished build, and the oldest unfinished build.
		var branches = {}; // branch -> branch
		var finished = {}; // branch -> builds
		var unfinished = {}; // branch -> builds
		_.forEach(rb.builds, function(b) {
			var br = b.branch;
			branches[br] = br;
			finished[br] = finished[br] || [];
			unfinished[br] = unfinished[br] || [];
			if (b.finish) {
				finished[br].push(b);
			} else {
				unfinished[br].push(b);
			}
		});
		var builds = [];
		_.forEach(branches, function(br) {
			var l = unfinished[br];
			if (l.length > 0) {
				l.sort(function(a, b) {
					return new Date(a.created).getTime() - new Date(b.created).getTime();
				});
				builds.push(l[0]);
			}
			l = finished[br];
			if (l.length > 0) {
				l.sort(function(a, b) {
					return new Date(b.finish).getTime() - new Date(a.finish).getTime();
				});
				builds.push(l[0]);
			}
		});
		rb.builds = builds;
	}

	$scope.$on('build', function(x, e) {
		var b = e.build;
		var repoName = e.repo_name;
		$timeout(function() {
			var rb = _.find($scope.repoBuilds, function(rb) {
				return rb.repo.name === repoName;
			});
			if (!rb) {
				console.log('build for unknown repo?', b, repoName);
				return;
			}
			var i = _.findIndex(rb.builds, {id: b.id});
			if(i >= 0) {
				rb.builds.splice(i, 1, b);
			} else {
				rb.builds.push(b);
			}
			buildsCleanup(rb);
		});
	});

	$scope.$on('removeBuild', function(x, e) {
		var build_id = e.build_id;
		$timeout(function() {
			// bug: when the most recent build is removed, this causes us to claim there are no builds (for the branch).
			for (var i = 0; i < $scope.repoBuilds.length; i++) {
				var rb = $scope.repoBuilds[i];
				rb.builds = _.filter(rb.builds, function(b) {  // jshint ignore:line
					return b.id !== build_id;
				});
			}
		});
	});

	$scope.youngestBuild = function(rb) {
		var tm;
		for(var i = 0; i < rb.builds.length; i++) {
			var b = rb.builds[i];
			if (!tm || b.start > tm) {
				tm = b.start;
			}
			if (!tm || b.created > tm) {
				tm = b.created;
			}
		}
		if (tm) {
			return new Date().getTime() - new Date(tm).getTime();
		}
		return Infinity;
	};

	$scope.clearRepoHomedirs = function() {
		return Msg.confirm('Are you sure?', function() {
			return api.clearRepoHomedirs();
		});
	};

	$scope.createLowPrioBuilds = function() {
		return api.createLowPrioBuilds();
	};

	$scope.newRepo = function() {
		return $uibModal.open({
			templateUrl: 'static/html/modals/new-repo.html',
			controller: function($scope, $uibModalInstance) {
				$scope.repo = {
					vcs: 'git',
					origin: '',
					name: '',
					checkout_path: ''
				};
				$scope.repoUID = true;
				$scope.nameAutoFill = true;
				$scope.checkoutpathAutoFill = true;
				$scope.$watch('repo.origin', function(v) {
					if (!v || $scope.vcs === 'command') {
						return;
					}
					var name = _.last(v.trim('/').split(/[:\/]/)).replace(/\.git$/, '');
					if ($scope.nameAutoFill) {
						$scope.repo.name = name;
					}
					if ($scope.checkoutpathAutoFill) {
						$scope.repo.checkout_path = name;
					}
				});

				$scope.create = function() {
					var repo = _.clone($scope.repo);
					repo.uid = $scope.repoUID ? 1 : null;
					return api.createRepo(repo)
					.then(function(repo) {
						$uibModalInstance.close();
						$location.path('/repo/' + repo.name);
					});
				};
				$scope.close = function() {
					$uibModalInstance.close();
				};
			}
		}).opened;
	};
});
