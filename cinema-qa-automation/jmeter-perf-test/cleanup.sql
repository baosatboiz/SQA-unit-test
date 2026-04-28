-- cleanup.sql
-- Idempotent — keeps qa_perf_user_* accounts, only clears their bookings/payments
DELETE FROM payments WHERE booking_id IN (
  SELECT b.id FROM bookings b JOIN users u ON b.user_id = u.id WHERE u.email LIKE 'qa_perf_%'
);
DELETE FROM bookings WHERE user_id IN (SELECT id FROM users WHERE email LIKE 'qa_perf_%');
