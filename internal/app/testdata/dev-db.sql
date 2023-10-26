CREATE TABLE IF NOT EXISTS talks (
  id serial,
  uuid varchar(255),
  title varchar(255)
);

INSERT
  INTO talks (uuid, title)
  VALUES ('testcontainers-integration-testing', 'Modern Integration Testing with Testcontainers')
  ON CONFLICT do nothing;

INSERT
  INTO talks (uuid, title)
  VALUES ('flight-of-the-gopher', 'A look at Go scheduler')
  ON CONFLICT do nothing;
