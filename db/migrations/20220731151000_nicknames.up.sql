alter table users ADD COLUMN nickname character varying;

--bun:split

create index nickname_no_special_chars on users(regexp_replace(nickname, '\W', '', 'g'));
