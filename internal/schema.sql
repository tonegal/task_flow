CREATE TABLE TASK_T (
    ID INTEGER PRIMARY KEY,
    TYPE SMALLINT NOT NULL,
    VALUE SMALLINT NOT NULL,
    STATE CHARACTER VARYING(10) NOT NULL,
    CREATION_TIME TIMESTAMP NOT NULL,
    LAST_UPDATE_TIME TIMESTAMP NOT NULL
);