select assert_schema_version(16);
insert into schema_upgrades (version) values (17);

update build set status = 'build' where status not in ('new', 'clone', 'build', 'success');

alter table build drop constraint build_status_check;
alter table build add constraint build_status_check check (status in ('new', 'clone', 'build', 'success', 'cancelled'));
