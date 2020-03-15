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
		return api.installGoToolchain(goversion, '')
		.then(function() {
			$scope.installed[goversion] = true;
		});
	};

	$scope.remove = function(goversion) {
		return api.removeGoToolchain(goversion)
		.then(function() {
			$scope.installed[goversion] = false;
		});
	};

	$scope.activate = function(goversion, shortname) {
		return api.activateGoToolchain(goversion, shortname)
		.then(function() {
			$scope.active[shortname] = goversion;
		});
	};
});
