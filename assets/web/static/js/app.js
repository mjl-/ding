// don't warn about "use strict"
/* jshint -W097 */
/* global api, $, window, angular, _, console */
'use strict';

var app = angular.module('app', [
	'templates',
	'ngRoute',
	'ui.bootstrap',
	'ui.bootstrap.modal',
	'ui.bootstrap.popover',
	'ui.bootstrap.tooltip',
	'ui.bootstrap.progressbar',
	'ui.bootstrap.tabs',
	'ui.bootstrap.datepickerPopup'
])
.run(function($rootScope, $window, $uibModal, $q, $timeout, Msg, Util) {
	api._wrapThenable = $q;

	$rootScope._app_version = api._sherpa.version;

	$rootScope.loading = false;
	$rootScope.loadingSaved = function() {
		$rootScope.loading = false;
		$('.x-loadingsaved').show().delay(1500).fadeOut('slow');
	};

	var password = window.localStorage.getItem('password') || '';
	var passwordPopupOpen = false;
	var openLogin = function() {
		if (passwordPopupOpen) {
			return;
		}
		passwordPopupOpen = true;

		$uibModal.open({
			templateUrl: 'static/html/modals/password.html',
			backdrop: 'static', // prevent closing popup
			keyboard: false, // also prevent closing with escape...
			controller: function($route, $scope, $uibModalInstance) {
				$scope.password = '';
				$scope.ok = function() {
					password = $scope.password;
					window.localStorage.setItem('password', password);
					passwordPopupOpen = false;
					$uibModalInstance.close();
					$route.reload();
					$rootScope.reconnect();
					return $q.resolve();
				};
			}
		});
	};
	$rootScope.password = function() {
		if (!password) {
			openLogin();
		}
		return password;
	};

	$rootScope.logout = function() {
		window.localStorage.removeItem('password');
		$rootScope.disconnect();
		window.location.reload();
		return $q.resolve();
	};

	var handleApiError = function(error) {
		console.log('Error loading page', error);
		var txt;
		if (error.code === 'user:badAuth') {
			openLogin();
			return;
		}
		if(_.has(error, 'message')) {
			txt = error.message;
		} else {
			txt = JSON.stringify(error);
		}
		Msg.alert('Error loading page: ' + txt);
	};

	$rootScope.$on('$routeChangeStart', function(event, next) {
		$rootScope.loading = true;
		$rootScope.breadcrumbs = [];
	});

	$rootScope.$on('$routeChangeSuccess', function() {
		$rootScope.loading = false;
	});

	$rootScope.$on('$routeChangeError', function(event, current, previous, rejection) {
		$rootScope.loading = false;
		handleApiError(rejection);
	});

	var eventSource;

	$rootScope.disconnect = function() {
		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}
	};
	$rootScope.reconnect = function() {
		if (!$rootScope.password()) {
			return;
		}
		if (eventSource) {
			eventSource.close();
		}
		eventSource = new window.EventSource('/events?password=' + encodeURIComponent($rootScope.password()));
		var kinds = [
			'repo',
			'removeRepo',
			'build',
			'removeBuild',
			'output'
		];
		_.forEach(kinds, function(kind) {
			eventSource.addEventListener(kind, function(e) {
				var m = JSON.parse(e.data);
				$rootScope.$broadcast(kind, m);
			});
		});
		eventSource.addEventListener('open', function(e) {
			$timeout(function() {
				$rootScope.sseError = '';
			});
		});
		eventSource.addEventListener('error', function(e) {
			// On page reload we will receive this error event. By actually sleeping a bit, we (likely) wait until the page is reloaded, and we won't see an error flash and a seeming layout shift in that case.
			$timeout(function() {
				$rootScope.sseError = true;
			}, 250);
		});
		return $q.resolve();
	};

	if (!!window.EventSource) {
		$rootScope.reconnect();
	} else {
		$rootScope.noSSE = true;
	}
});
// don't warn about "use strict"
/* jshint -W097 */
/* global app, api */
'use strict';

app.config(function($routeProvider, $uibTooltipProvider) {
	$uibTooltipProvider.options({
		placement: 'right',
		popupDelay: 500, // ms
		appendToBody: true
	});

	$routeProvider
	.when('/', {
		templateUrl: 'static/html/index.html',
		controller: 'Index',
		resolve: {
			repoBuilds: function($rootScope) {
				return api.repoBuilds($rootScope.password());
			}
		}
	})
	.when('/repo/:repoName/', {
		templateUrl: 'static/html/repo.html',
		controller: 'Repo',
		resolve: {
			repo: function($rootScope, $route) {
				return api.repo($rootScope.password(), $route.current.params.repoName);
			},
			builds: function($rootScope, $route) {
				return api.builds($rootScope.password(), $route.current.params.repoName);
			}
		}
	})
	.when('/repo/:repoName/build/:buildId/', {
		templateUrl: 'static/html/build.html',
		controller: 'Build',
		resolve: {
			repo: function($rootScope, $route) {
				return api.repo($rootScope.password(), $route.current.params.repoName);
			},
			build: function($rootScope, $route) {
				return api.build($rootScope.password(), $route.current.params.repoName, parseInt($route.current.params.buildId));
			}
		}
	})
	.when('/repo/:repoName/release/:buildId/', {
		templateUrl: 'static/html/release.html',
		controller: 'Release',
		resolve: {
			repo: function($rootScope, $route) {
				return api.repo($rootScope.password(), $route.current.params.repoName);
			},
			build: function($rootScope, $route) {
				return api.release($rootScope.password(), $route.current.params.repoName, parseInt($route.current.params.buildId));
			}
		}
	})
	.when('/gotoolchains/', {
		templateUrl: 'static/html/gotoolchains.html',
		controller: 'Gotoolchains',
		resolve: {
			released: function($rootScope) {
				return api.listReleasedGoToolchains($rootScope.password());
			},
			installed: function($rootScope) {
				return api.listInstalledGoToolchains($rootScope.password());
			}
		}
	})
	.when('/help/', {
		templateUrl: 'static/html/help.html',
		controller: function($rootScope, Util) {
			$rootScope.breadcrumbs = Util.crumbs([
				Util.crumb('/help/', 'Help')
			]);
		}
	})
	.otherwise({
		templateUrl: 'static/html/404.html'
	});
});
// don't warn about "use strict"
/* jshint -W097 */
/* global app, api, _, console, window, document, $ */
'use strict';

app.controller('Build', function($scope, $rootScope, $q, $location, $timeout, Msg, Util, repo, build) {
	$rootScope.breadcrumbs = Util.crumbs([
		Util.crumb('repo/' + repo.name, 'Repo ' + repo.name),
		Util.crumb('build/' + build.id + '/', 'Build ' + build.id)
	]);

	$scope.repo = repo;
	$scope.build = build;
	$scope.build.steps = $scope.build.steps || [];
	$scope.canceled = false;

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
			var step = _.find($scope.build.steps, {name: e.step});
			if (!step) {
				step = {
					name: e.step,
					output: '',
					// nsec: 0,
					_start: new Date().getTime()
				};
				$scope.build.steps.push(step);
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
		return api.cancelBuild($rootScope.password(), repo.name, $scope.build.id)
		.then(function() {
			$scope.canceled = true;
		});
	};
});
// don't warn about "use strict"
/* jshint -W097 */
/* global app, api, _, console */
'use strict';

app.controller('Gotoolchains', function($scope, $rootScope, $q, $uibModal, $location, $timeout, Msg, Util, released, installed) {
	$rootScope.breadcrumbs = Util.crumbs([
		Util.crumb('/gotoolchains/', 'Go toolchains')
	]);

	$scope.released = released;
	$scope.installed = {};
	_.forEach(installed[0], function(goversion) {
		$scope.installed[goversion] = true;
	});
	$scope.active = installed[1];

	$scope.install = function(goversion) {
		return api.installGoToolchain($rootScope.password(), goversion, '')
		.then(function() {
			$scope.installed[goversion] = true;
		});
	};

	$scope.remove = function(goversion) {
		return api.removeGoToolchain($rootScope.password(), goversion)
		.then(function() {
			$scope.installed[goversion] = false;
		});
	};

	$scope.activate = function(goversion, shortname) {
		return api.activateGoToolchain($rootScope.password(), goversion, shortname)
		.then(function() {
			$scope.active[shortname] = goversion;
		});
	};
});
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
			return api.clearRepoHomedirs($rootScope.password());
		});
	};

	$scope.createLowPrioBuilds = function() {
		return api.createLowPrioBuilds($rootScope.password());
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
					return api.createRepo($rootScope.password(), repo)
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
// don't warn about "use strict"
/* jshint -W097 */
/* global app, api */
'use strict';

app.controller('Release', function($scope, $rootScope, $q, $location, Msg, Util, repo, build) {
	$rootScope.breadcrumbs = Util.crumbs([
		Util.crumb('repo/' + repo.name, 'Repo ' + repo.name),
		Util.crumb('release/' + build.id + '/', 'Release ' + build.id)
	]);

	$scope.repo = repo;
	$scope.build = build;
});
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
					age = Math.floor(sec / 60) + 'm';
				} else if (sec < 48*3600) {
					age = Math.floor(sec / 3600) + 'h';
					wait = 60 * 1000;
				} else if (sec < 21*24*3600) {
					age = Math.floor(sec / (24*3600)) + 'd';
					wait = 3600 * 1000;
				} else {
					age = Math.floor(sec / (7*24*3600)) + 'w';
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
/* jshint -W097 */ // for "use strict"
/* global app */
'use strict';

app
.directive('btn', function() {
	return {
		priority: 10,
		link: function(scope, element, attrs) {
			var $a = element.find('a');
			if($a.length) {
				element = $a;
			}
			element.addClass('btn');
			if(attrs.btn) {
				var l = attrs.btn.split(' ');
				for (var i = 0; i < l.length; i++) {
					element.addClass('btn-'+l[i]);
				}
			}
		}
	};
});
/* jshint -W097 */ // for "use strict"
/* global app, console */
'use strict';

app
.directive('buildStatus', function() {
	return {
		restrict: 'E',
		template: '<span><span class="label" ng-class="{\'label-primary\': released && status === \'success\', \'label-success\': !released && finish && status === \'success\', \'label-danger\': finish && status !== \'success\', \'label-default\': !finish}" style="margin-right: 0.25rem">{{ status }}{{ lowprio ? "↓" : "" }}</span><span class="fa fa-cog fa-spin" ng-if="!finish && status !== \'new\'" style="vertical-align: middle"></span></span>',
		scope: {
			'status': '=',
			'finish': '=',
			'released': '=',
			'lowprio': '='
		}
	};
});
/* jshint -W097 */ // for "use strict"
/* global app, window, console */
'use strict';

app
.directive('buildtime', function($timeout) {
	return {
		restrict: 'E',
		template: '<span><span ng-if="start">{{ elapsed.toFixed((elapsed < 10 || finish) ? 1 : 0) }}s</span><span ng-if="!finish">...</span></span>',
		scope: {
			'start': '=',
			'finish': '='
		},
		link: function(scope, element) {
			var timeoutID;
			var scheduleUpdate = function() {
				var spentMS = new Date().getTime() - new Date(scope.start).getTime();
				var wait = spentMS < 10*1000 ? 100 : 1000;
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
/* jshint -W097 */ // for "use strict"
/* global app */
'use strict';

app
.directive('clickStopPropagation', function() {
	return {
		restrict: 'A',
		link: function(scope, element) {
			element.bind('click', function(e) {
				e.stopPropagation();
			});
		}
	};
});
/* jshint -W097 */ // for "use strict"
/* global app, console */
'use strict';

app
.directive('coverage', function() {
	return {
		restrict: 'E',
		template: '<a ng-if="build.finish && build.status === &quot;success&quot;" ng-href="{{ build.coverage_report_file ? &quot;/dl/file/&quot; + repo.name + &quot;/&quot; + build.id + &quot;/&quot; + build.coverage_report_file : &quot;&quot;}}"><span ng-if="build.coverage === null" class="label label-danger"><i class="fa fa-file-code-o" ng-if="build.coverage_report_file"></i> none</span><span ng-if="build.coverage !== null" class="label" ng-class="{&quot;label-warning&quot;: build.coverage &lt; 60, &quot;label-info&quot;: build.coverage &gt;= 60 &amp;&amp; build.coverage &lt; 85, &quot;label-success&quot;: build.coverage &gt;= 85}"><i class="fa fa-file-code-o" ng-if="build.coverage_report_file"></i> {{ build.coverage.toFixed(0) }}% </span></a>',
		scope: {
			'build': '=',
			'repo': '='
		}
	};
});
/* jshint -W097 */ // for "use strict"
/* global app */
'use strict';

app
.directive('enter', function() {
	return {
		restrict: 'A',
		link: function(scope, element, attrs) {
			element.bind('keydown keypress', function(e) {
				var key = 'which' in e ? e.which : e.keyCode;
				if(key === 13) {
					// enter
					scope.$apply(function() {
						scope.$eval(attrs.enter);
					});
					e.preventDefault();
				}
			});
		}
	};
});
/* jshint -W097 */ // for "use strict"
/* global app, console */
'use strict';

app
.directive('filesize', function() {
	return {
		restrict: 'E',
		template: '{{ size === 0 ? "-" : (size / (1024*1024)).toFixed(1)+"m" }}',
		scope: {
			'size': '='
		}
	};
});
/* jshint -W097 */ // for "use strict"
/* global app, $, document */
'use strict';

app
.directive('icon', function() {
	return {
		priority: 20,
		link: function(scope, element, attrs) {
			var $icon = $('<i class="fa"></i>');
			$icon.addClass('fa-'+attrs.icon);
			var $a = element.find('a');
			if($a.length) {
				element = $a;
			}
			element.prepend($(document.createTextNode(' ')));
			element.prepend($icon);
		}
	};
});
/* jshint -W097 */ // for "use strict"
/* global app */
'use strict';

app
.directive('linkDisabled', function() {
	return {
		restrict: 'A',
		scope: {
			'linkDisabled': '='
		},
		link: function(scope, element) {
			scope.$watch('linkDisabled', function(v) {
				if(v) {
					element.attr('disabled', 'disabled');
				} else {
					element.removeAttr('disabled');
				}
			});
			element.bind('click', function(e) {
				if(scope.linkDisabled) {
					e.stopPropagation();
					e.preventDefault();
				}
			});
		}
	};
});
// don't warn about "use strict"
/* jshint -W097 */
/* global app */
'use strict';

app.directive('loadingClick', function($rootScope, Msg) {
	return {
		restrict: 'A',
		link: function(scope, element, attrs) {
			element.on('click', function(e) {
				e.preventDefault();
				e.stopPropagation();

				if (element.attr('disabled')) {
					return;
				}

				scope.$apply(function() {
					$rootScope.loading = true;
				});

				scope.$eval(attrs.loadingClick)
				.then(function() {
					$rootScope.loading = false;
				}, function(error) {
					$rootScope.loading = false;
					if(error) {
						Msg.alert(error.message);
					}
				});
			});
		}
	};
})
.directive('savingClick', function($rootScope, Msg) {
	return {
		restrict: 'A',
		link: function(scope, element, attrs) {
			element.on('click', function(e) {
				e.preventDefault();
				e.stopPropagation();

				if (element.attr('disabled')) {
					return;
				}

				scope.$apply(function() {
					$rootScope.loading = true;
				});

				scope.$eval(attrs.savingClick)
				.then(function() {
					$rootScope.loadingSaved();
				}, function(error) {
					$rootScope.loading = false;
					if(error) {
						Msg.alert(error.message);
					}
				});
			});
		}
	};
})
.directive('loadingSubmit', function($rootScope, Msg) {
	return {
		restrict: 'A',
		link: function(scope, element, attrs) {
			element.on('submit', function(e) {
				e.preventDefault();
				e.stopPropagation();

				if (element.attr('disabled')) {
					return;
				}

				scope.$apply(function() {
					$rootScope.loading = true;
				});

				scope.$eval(attrs.loadingSubmit)
				.then(function() {
					$rootScope.loading = false;
				}, function(error) {
					$rootScope.loading = false;
					if(error) {
						Msg.alert(error.message);
					}
				});
			});
		}
	};
})
.directive('savingSubmit', function($rootScope, Msg) {
	return {
		restrict: 'A',
		link: function(scope, element, attrs) {
			element.on('submit', function(e) {
				e.preventDefault();
				e.stopPropagation();

				if (element.attr('disabled')) {
					return;
				}

				scope.$apply(function() {
					$rootScope.loading = true;
				});

				scope.$eval(attrs.savingSubmit)
				.then(function() {
					$rootScope.loadingSaved();
				}, function(error) {
					$rootScope.loading = false;
					if(error) {
						Msg.alert(error.message);
					}
				});
			});
		}
	};
});
/* jshint -W097 */ // for "use strict"
/* global app, location */
'use strict';

app
.directive('rowClick', function() {
	return {
		restrict: 'A',
		link: function(scope, element, attrs) {
			element.addClass('clickrow');
			element.on('click', function(e) {
				e.preventDefault();
				e.stopPropagation();

				location.href = attrs.rowClick;
			});
		}
	};
});
/* jshint -W097 */ // for "use strict"
/* global app, console */
'use strict';

app
.directive('timespent', function() {
	return {
		restrict: 'E',
		template: '{{ (nsec / (1000 * 1000)).toFixed(0) }} ms',
		scope: {
			'nsec': '='
		}
	};
});
// don't warn about "use strict"
/* jshint -W097 */
/* global app */
'use strict';

app
.filter('basename', function() {
	return function(text) {
		var t = text.split('/');
		return t[t.length-1];
	};
});
// don't warn about "use strict"
/* jshint -W097 */
/* global app, $, angular, _, window */
'use strict';

app
.filter('titleize', function() {
	return function(text) {
		return _.capitalize(text);
	};
});
// don't warn about "use strict"
/* jshint -W097 */
/* global app, _ */
'use strict';

app.service('Msg', function($q, $window, $rootScope, $uibModal, $sce) {
	this.alert = function(message) {
		return $uibModal.open({
			templateUrl: 'static/html/modals/alert.html',
			controller: function($scope, $uibModalInstance) {
				$scope.title = 'Fout!';
				$scope.message = message;
				$scope.alertClass = 'danger';

				$scope.close = function() {
					$uibModalInstance.close();
				};
			}
		}).opened;
	};

	this.dialog = function(message, alertClass) {
		return $uibModal.open({
			templateUrl: 'static/html/modals/alert.html',
			controller: function($scope, $uibModalInstance) {
				$scope.title = {
					danger: 'Fout',
					warning: 'Waarschuwing',
					info: 'Geslaagd'
				}[alertClass];
				$scope.message = message;
				$scope.alertClass = alertClass;

				$scope.close = function() {
					$uibModalInstance.close();
				};
			}
		}).opened;
	};

	this.confirm = function confirm(message, handle) {
		if (!message) {
			message = 'Weet je het zeker?';
		}

		return $uibModal.open({
			templateUrl: 'static/html/modals/confirm.html',
			controller: function($scope, $uibModalInstance) {
				$scope.message = message;

				$scope.confirm = function() {
					$uibModalInstance.close();
					return handle();
				};

				$scope.dismiss = function() {
					$uibModalInstance.dismiss();
				};
			}
		}).opened;
	};

	this.linkPost = function linkPost(url, message, action, pairs) {
		$sce.trustAsUrl(url);

		return $uibModal.open({
			templateUrl: 'static/html/modals/link-post.html',
			controller: function($scope, $uibModalInstance) {
				$scope.url = url;
				$scope.message = message;
				$scope.action = action;
				$scope.pairs = pairs;
				$scope.close = function() {
					$uibModalInstance.close();
				};
			}
		}).opened;
	};

	this.link = function link(url, action, message) {
		$sce.trustAsUrl(url);

		return $uibModal.open({
			templateUrl: 'static/html/modals/link.html',
			controller: function($scope, $uibModalInstance) {
				$scope.url = url;
				$scope.message = message;
				$scope.action = action;

				$scope.close = function() {
					$uibModalInstance.close();
				};
			}
		}).opened;
	};
});
// don't warn about "use strict"
/* jshint -W097 */
/* global app, api, console, window */
'use strict';

app.factory('Util', function($q, $window, $rootScope, $uibModal) {

	function readFile($file) {
		if($file.length !== 1) {
			return $q.reject({message: 'Bad input type=file'});
		}
		var files = $file[0].files;
		if(files.length != 1) {
			return $q.reject({message: 'Need exactly 1 file.'});
		}
		var file = files[0];

		var defer = $q.defer();
		var fr = new window.FileReader();
		fr.onload = function(e) {
			defer.resolve(e.target.result);
		};
		fr.onerror = function(e) {
			console.log('error', e);
			defer.reject({message: 'Error reading file'});
		};
		fr.readAsDataURL(file);
		return defer.promise;
	}

	function crumb(path, label) {
		return {path: path, label: label};
	}

	function crumbs(l) {
		for(var i = 1; i < l.length; i++) {
			l[i].path = l[i-1].path+'/'+l[i].path;
		}
		return l;
	}

	return {
		readFile: readFile,
		crumb: crumb,
		crumbs: crumbs
	};
});
