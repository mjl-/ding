select assert_schema_version(12);
insert into schema_upgrades (version) values (13);

alter table build add column created timestamptz not null default now();
update build set created=start;
alter table build alter column start drop not null;
alter table build alter column start drop default;

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
