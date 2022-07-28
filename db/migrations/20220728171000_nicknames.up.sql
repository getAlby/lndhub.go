alter table users ADD COLUMN nickname character varying;

--bun:split

create unique index nickname_no_special_chars on users(regexp_replace(nickname, '\W', '', 'g'));
