create table if not exists message
(
    id         int auto_increment
    primary key,
    message    text                               null,
    status     varchar(20)                        null,
    details    text                               null,
    created_at datetime default CURRENT_TIMESTAMP not null,
    updated_at datetime                           null on update CURRENT_TIMESTAMP
    );

