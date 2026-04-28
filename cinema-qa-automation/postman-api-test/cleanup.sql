-- Idempotent cleanup for Postman API tests
-- Run after each test session to remove all qa_api_* test data

DELETE FROM payments
WHERE booking_id IN (
  SELECT b.id
  FROM bookings b
  JOIN users u ON b.user_id = u.id
  WHERE u.email LIKE 'qa_api_%'
);

DELETE FROM bookings
WHERE user_id IN (
  SELECT id FROM users WHERE email LIKE 'qa_api_%'
);

DELETE FROM users
WHERE email LIKE 'qa_api_%';
