select assert_schema_version(17);
insert into schema_upgrades (version) values (18);

alter table repo add column default_branch text not null default '';
update repo set default_branch='master' where vcs='git';
update repo set default_branch='default' where vcs='bitbucket';
