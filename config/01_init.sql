-- initialiaze the db with a test user 
INSERT INTO users (
    first_name,
    last_name,
    username,
    password,
    income,
    income_rate
) VALUES ('test', 'user', 'testuser', '$2a$10$AQOLxQTZS/V3XS/3qFkUxe3RQ6UqqOW/a2zuxJAN0Nhpe5zUVnoSu', 1000, 'weekly');