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
