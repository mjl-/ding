select assert_schema_version(14);
insert into schema_upgrades (version) values (15);

alter table repo add home_disk_usage bigint not null default 0;
alter table build add home_disk_usage_delta bigint not null default 0;

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
