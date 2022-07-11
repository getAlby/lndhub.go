alter table users ADD COLUMN nickname character varying NOT NULL;

--bun:split

create unique index nickname_no_special_chars on users(regexp_replace(nickname, '\W', '', 'g'));

--bun:split

create UNIQUE INDEX change_nickname_on_collision on users(login, password);