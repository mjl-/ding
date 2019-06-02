// don't warn about "use strict"
/* jshint -W097 */
/* global app, api, _, console */
'use strict';

app.controller('Index', function($scope, $rootScope, $q, $uibModal, $location, $timeout, Msg, Util, repoBuilds) {
	$rootScope.breadcrumbs = Util.crumbs([]);

	var removeBuild = function(l, buildID) {
		var i = _.findIndex(l, {id: buildID});
		if(i >= 0) {
			l.splice(i, 1);
		}
	};

	// Sort builds by finish time descending first (if set), and id descending second.
	var sortBuilds = function(l) {
		l.sort(function(a, b) {
			if (a.finish && b.finish) {
				return new Date(b.finish).getTime() - new Date(a.finish).getTime();
			}
			return b.id-a.id;
		});
	};

	// We transform repoBuilds into repoBranchBuilds consisting of a repo and per
	// branch a  list of finished and unfinished builds.
	// After a change, we generate $scope.repoBuilds with the information we want
	// displayed. AngularJS templates cannot easily generate rows from nested data...
	var repoBranchBuilds = _.map(repoBuilds, function(rb) {
		var rbb = {
			repo: rb.repo,
			branchBuilds: {}
		};
		_.forEach(rb.builds, function(b) {
			var bb = rbb.branchBuilds[b.branch];
			if (!bb) {
				bb = rbb.branchBuilds[b.branch] = {
					finished: [],
					unfinished: []
				};
			}
			if (b.finish) {
				bb.finished.push(b);
				sortBuilds(bb.finished);
			} else {
				bb.unfinished.push(b);
				sortBuilds(bb.unfinished);
			}
		});
		return rbb;
	});

	var makeRepoBuilds = function() {
		$scope.repoBuilds = _.map(repoBranchBuilds, function(rbb) {
			var builds = [];
			_.forEach(rbb.branchBuilds, function(bb, branch) {
				_.forEach(bb.unfinished, function(b) {
					builds.push(b);
				});
				if (bb.finished.length > 0) {
					builds.push(bb.finished[0]);
				}
			});
			return {
				repo: rbb.repo,
				builds: builds
			};
		});

		var timestamp = function(s) {
			if (!s) {
				return 0;
			}
			return new Date(s).getTime();
		};

		var buildTimestamp = function(b) {
			var ts = Math.max(timestamp(b.created), timestamp(b.start), timestamp(b.finish));
			return ts;
		};

		var mostRecent = function(l) {
			var ts = 0;
			_.forEach(l, function(b) {
				ts = Math.max(ts, buildTimestamp(b));
			});
			return ts;
		};

		// Sort repo's by most recent activity (most recent of created,start,finish of build).
		$scope.repoBuilds.sort(function(a, b) {
			return mostRecent(b.builds)-mostRecent(a.builds);
		});
	};
	makeRepoBuilds();

	$scope.$on('repo', function(x, e) {
		$timeout(function() {
			var r = e.repo;
			var rbb = _.find(repoBranchBuilds, function(rbb) {
				return rbb.repo.name === r.name;
			});
			if (rbb) {
				rbb.repo = r;
			} else {
				repoBranchBuilds.push({
					repo: r,
					branchBuilds: {}
				});
			}
			makeRepoBuilds();
		});
	});

	$scope.$on('removeRepo', function(x, e) {
		$timeout(function() {
			repoBranchBuilds = _.filter(repoBranchBuilds, function(rbb) {
				return rbb.repo.name !== e.repo_name;
			});
			makeRepoBuilds();
		});
	});

	$scope.$on('build', function(x, e) {
		var b = e.build;
		var repoName = e.repo_name;
		$timeout(function() {
			var rbb = _.find(repoBranchBuilds, function(rbb) {
				return rbb.repo.name === repoName;
			});
			if (!rbb) {
				console.log('build for unknown repo?', b, repoName);
				return;
			}
			var bb = rbb.branchBuilds[b.branch];
			removeBuild(bb.finished, b.id);
			removeBuild(bb.unfinished, b.id);
			if (b.finish) {
				bb.finished.push(b);
				sortBuilds(bb.finished);
			} else {
				bb.unfinished.push(b);
				sortBuilds(bb.unfinished);
			}
			makeRepoBuilds();
		});
	});

	$scope.$on('removeBuild', function(x, e) {
		var build_id = e.build_id;
		$timeout(function() {
			_.forEach(repoBranchBuilds, function(rbb) {
				_.forEach(rbb.branchBuilds, function(bb) {
					removeBuild(bb.finished, build_id);
					removeBuild(bb.unfinished, build_id);
				});
			});
			makeRepoBuilds();
		});
	});

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
