/* jshint -W097 */ // for "use strict"
/* global app, console */
'use strict';

app
.directive('coverage', function() {
	return {
		restrict: 'E',
		template: '<a ng-if="build.finish && build.status === &quot;success&quot;" ng-href="{{ build.coverage_report_file ? &quot;/dl/file/&quot; + repo.name + &quot;/&quot; + build.id + &quot;/&quot; + build.coverage_report_file : &quot;&quot;}}"><span ng-if="build.coverage === null" class="label label-danger"><i class="fa fa-file-code-o" ng-if="build.coverage_report_file"></i> none</span><span ng-if="build.coverage !== null" class="label" ng-class="{&quot;label-warning&quot;: build.coverage &lt; 50, &quot;label-info&quot;: build.coverage &gt;= 50 &amp;&amp; build.coverage &lt; 90, &quot;label-success&quot;: build.coverage &gt;= 90}"><i class="fa fa-file-code-o" ng-if="build.coverage_report_file"></i> {{ build.coverage.toFixed(0) }}% </span></a>',
		scope: {
			'build': '=',
			'repo': '='
		}
	};
});
