select assert_schema_version(11);
insert into schema_upgrades (version) values (12);

alter table build add column coverage real;
alter table build add column coverage_report_file text not null default '';

-- Must recreate view after adding/removing columns.
drop view build_with_result;
create view build_with_result as
select
	build.*,
	array_remove(array_agg(result.*), null) as results
from build
left join result on build.id = result.build_id
group by build.id
;
