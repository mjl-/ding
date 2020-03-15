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
