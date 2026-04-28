package cinema.utils;

import java.sql.*;

public class DBHelper {
    private static Connection getConn() throws SQLException {
        return DriverManager.getConnection(
            ConfigReader.get("db.url"),
            ConfigReader.get("db.user"),
            ConfigReader.get("db.password"));
    }

    public static boolean userExists(String email) {
        String sql = "SELECT 1 FROM users WHERE email = ?";
        try (Connection c = getConn(); PreparedStatement ps = c.prepareStatement(sql)) {
            ps.setString(1, email);
            try (ResultSet rs = ps.executeQuery()) { return rs.next(); }
        } catch (SQLException e) { throw new RuntimeException(e); }
    }

    public static int countBookings(String email) {
        String sql = """
            SELECT COUNT(*) FROM bookings b
            JOIN users u ON b.user_id = u.id
            WHERE u.email = ?
            """;
        try (Connection c = getConn(); PreparedStatement ps = c.prepareStatement(sql)) {
            ps.setString(1, email);
            try (ResultSet rs = ps.executeQuery()) {
                if (rs.next()) return rs.getInt(1);
                return 0;
            }
        } catch (SQLException e) { throw new RuntimeException(e); }
    }

    public static String getLatestBookingStatus(String email) {
        String sql = """
            SELECT b.status FROM bookings b
            JOIN users u ON b.user_id = u.id
            WHERE u.email = ?
            ORDER BY b.created_at DESC LIMIT 1
            """;
        try (Connection c = getConn(); PreparedStatement ps = c.prepareStatement(sql)) {
            ps.setString(1, email);
            try (ResultSet rs = ps.executeQuery()) {
                if (rs.next()) return rs.getString(1);
                return null;
            }
        } catch (SQLException e) { throw new RuntimeException(e); }
    }

    public static void cleanup() {
        String prefix = ConfigReader.get("test.data.email.prefix") + "%";
        try (Connection c = getConn()) {
            c.setAutoCommit(false);
            try (PreparedStatement ps1 = c.prepareStatement(
                    "DELETE FROM payments WHERE booking_id IN (SELECT b.id FROM bookings b JOIN users u ON b.user_id=u.id WHERE u.email LIKE ?)");
                 PreparedStatement ps2 = c.prepareStatement(
                    "DELETE FROM bookings WHERE user_id IN (SELECT id FROM users WHERE email LIKE ?)");
                 PreparedStatement ps3 = c.prepareStatement(
                    "DELETE FROM users WHERE email LIKE ?")) {
                ps1.setString(1, prefix); ps1.executeUpdate();
                ps2.setString(1, prefix); ps2.executeUpdate();
                ps3.setString(1, prefix); ps3.executeUpdate();
            }
            c.commit();
        } catch (SQLException e) { throw new RuntimeException("Cleanup failed", e); }
    }
}
