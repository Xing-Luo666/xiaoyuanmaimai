-- 更新 testcorp 密码为 corp123456
UPDATE u_corp_user
SET password = '$2a$10$FW61Ao9O7E3e/MPqZ.pi9OfQKf9lZYTefDOycAr6tcAXppQYKFsg6',
    status = 1,
    audit_status = 2
WHERE username = 'testcorp';

-- 同步更新init_tables.sql中的密码hash，便于后续重新初始化
