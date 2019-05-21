select assert_schema_version(10);
insert into schema_upgrades (version) values (11);

alter table repo add column uid int;
