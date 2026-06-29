UPDATE sys_user SET password='$2a$10$OWqmV9735md2JIra29db9.aTlaWa3ES/QVnESelz/AyUX6eBSDsGq' WHERE username='admin';
UPDATE u_user SET password='$2a$10$OWqmV9735md2JIra29db9.aTlaWa3ES/QVnESelz/AyUX6eBSDsGq';
UPDATE u_corp_user SET password='$2a$10$OWqmV9735md2JIra29db9.aTlaWa3ES/QVnESelz/AyUX6eBSDsGq';
SELECT username, LEFT(password, 30) AS pwd_prefix FROM sys_user WHERE username='admin';
