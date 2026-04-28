package cinema.tests;

import cinema.base.BaseTest;
import cinema.pages.BookingHistoryPage;
import cinema.pages.LoginPage;
import cinema.pages.MovieListPage;
import cinema.pages.PaymentPage;
import cinema.pages.SeatSelectionPage;
import cinema.pages.ShowtimesModalPage;
import cinema.utils.ConfigReader;
import cinema.utils.DBHelper;
import cinema.utils.ExcelReader;
import org.testng.Assert;
import org.testng.annotations.BeforeClass;
import org.testng.annotations.Test;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.Map;

/**
 * UI-08, UI-09, UI-10 — Booking Flow tests.
 *
 * BookingData sheet columns:
 *   tcId | email | password | movieIndex | showtimeIndex | seatStrategy | expectedResult
 *
 * UI-08 (priority 1): Full happy-path booking to payment page.
 *   Login → Movies → click first movie → select showtime → seat selection →
 *   confirm → reach PaymentPage → DB verify booking exists.
 *
 * UI-09 (priority 2, depends on UI-08): Same showtime, verify booked seats are disabled.
 *
 * UI-10 (priority 3, depends on UI-08): Open BookingHistoryPage, assert count >= 1.
 *
 * Note: PaymentPage.waitForLoad() asserts the "Thanh toán" heading is visible — this
 * is sufficient to confirm the booking was created and navigated to the payment screen
 * without completing the actual payment (QR/crypto flows require external action).
 */
public class BookingFlowTest extends BaseTest {

    private static final String DATA_FILE    = "test-data/ClientFlowData.xlsx";
    private static final String BOOKER_EMAIL    = "qa_test_booker@example.com";
    private static final String BOOKER_PASSWORD = "Pass@1234";

    /** Zero-based movie index used across UI-08, UI-09. */
    private static final int MOVIE_INDEX    = 0;
    /** Zero-based showtime index used across UI-08, UI-09. */
    private static final int SHOWTIME_INDEX = 0;

    /**
     * Pre-condition: register qa_test_booker@example.com via API.
     * Skips silently if the user already exists in DB.
     */
    @BeforeClass(alwaysRun = true, dependsOnMethods = "setUp")
    public void seedBooker() {
        try {
            if (DBHelper.userExists(BOOKER_EMAIL)) {
                System.out.println("[Seed] Booker user already in DB, skipping.");
                return;
            }
        } catch (Exception e) {
            System.err.println("[Seed] DB check failed (DB may be offline): " + e.getMessage());
        }
        try {
            HttpClient client = HttpClient.newHttpClient();
            String body = String.format(
                "{\"email\":\"%s\",\"password\":\"%s\",\"confirmPassword\":\"%s\"," +
                "\"firstName\":\"Booker\",\"lastName\":\"QA\",\"address\":\"789 Booker Rd\"}",
                BOOKER_EMAIL, BOOKER_PASSWORD, BOOKER_PASSWORD);
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(ConfigReader.get("api.base.url") + "/api/v1/auth/register"))
                    .header("Content-Type", "application/json")
                    .POST(HttpRequest.BodyPublishers.ofString(body))
                    .build();
            HttpResponse<String> response =
                    client.send(request, HttpResponse.BodyHandlers.ofString());
            System.out.println("[Seed] Seeded booker via API. Status: " + response.statusCode()
                    + " Body: " + response.body());
        } catch (Exception e) {
            System.err.println("[Seed] Warning — could not seed booker: " + e.getMessage());
        }
    }

    // ── UI-08: Happy Path ─────────────────────────────────────────────────────

    /**
     * UI-08: Full booking happy path.
     * Reads the first PASS row from BookingData for configuration (email, password, movieIndex, showtimeIndex).
     * Falls back to hard-coded booker credentials if the sheet cannot supply them.
     */
    @Test(priority = 1,
          description = "UI-08: Happy-path booking — Login → Movie → Showtime → Seat → Payment")
    public void happyPath_UI08() {
        // Resolve test parameters from BookingData sheet (first PASS row)
        String email        = BOOKER_EMAIL;
        String password     = BOOKER_PASSWORD;
        int    movieIdx     = MOVIE_INDEX;
        int    showtimeIdx  = SHOWTIME_INDEX;

        try {
            Object[][] rows = ExcelReader.toDataProvider(DATA_FILE, "BookingData");
            for (Object[] row : rows) {
                @SuppressWarnings("unchecked")
                Map<String, String> data = (Map<String, String>) row[0];
                if ("PASS".equalsIgnoreCase(data.get("expectedResult"))) {
                    email      = data.get("email");
                    password   = data.get("password");
                    movieIdx   = parseIndex(data.get("movieIndex"),   MOVIE_INDEX);
                    showtimeIdx = parseIndex(data.get("showtimeIndex"), SHOWTIME_INDEX);
                    break;
                }
            }
        } catch (Exception e) {
            System.err.println("[UI-08] Could not read BookingData, using defaults: " + e.getMessage());
        }

        // 1. Login
        LoginPage lp = new LoginPage(driver).open();
        lp.enterEmail(email).enterPassword(password).submit();
        Assert.assertTrue(lp.isLoggedIn(),
                "UI-08: Login failed for " + email +
                " — ensure user is email-verified in the test DB.");

        // 2. Navigate to movies and click target card
        MovieListPage mlp = new MovieListPage(driver).open();
        mlp.clickCardAt(movieIdx);

        // 3. Showtime modal
        ShowtimesModalPage modal = new ShowtimesModalPage(driver);
        Assert.assertTrue(modal.isOpen(), "UI-08: ShowtimesModal did not open.");
        modal.selectShowtime(showtimeIdx);
        modal.confirmBooking();

        // 4. Seat selection
        SeatSelectionPage ssp = new SeatSelectionPage(driver);
        ssp.selectFirstAvailable();
        ssp.confirm();

        // 5. Payment page — just verify we arrived (no actual payment)
        PaymentPage pp = new PaymentPage(driver);
        pp.waitForLoad();

        String currentUrl = driver.getCurrentUrl();
        Assert.assertTrue(
                currentUrl != null && currentUrl.contains("/booking"),
                "UI-08: Expected to reach payment page (URL containing '/booking') but got: " + currentUrl);

        // 6. DB verification
        try {
            int bookingCount = DBHelper.countBookings(email);
            Assert.assertTrue(bookingCount >= 1,
                    "UI-08: Expected >= 1 booking in DB for " + email + " but found " + bookingCount);
        } catch (Exception e) {
            System.err.println("[UI-08] DB verification skipped (DB offline?): " + e.getMessage());
        }
    }

    // ── UI-09: Booked seat disabled ───────────────────────────────────────────

    /**
     * UI-09: After UI-08, revisit the same showtime and assert that at least one seat
     * is shown as booked (cursor-not-allowed / bg-gray-500).
     */
    @Test(priority = 2,
          dependsOnMethods = "happyPath_UI08",
          description = "UI-09: Previously booked seat should be disabled on seat selection")
    public void bookedSeatDisabled_UI09() {
        // 1. Login
        LoginPage lp = new LoginPage(driver).open();
        lp.enterEmail(BOOKER_EMAIL).enterPassword(BOOKER_PASSWORD).submit();
        Assert.assertTrue(lp.isLoggedIn(),
                "UI-09: Login failed for " + BOOKER_EMAIL);

        // 2. Navigate to the same movie & showtime
        MovieListPage mlp = new MovieListPage(driver).open();
        mlp.clickCardAt(MOVIE_INDEX);

        ShowtimesModalPage modal = new ShowtimesModalPage(driver);
        Assert.assertTrue(modal.isOpen(), "UI-09: ShowtimesModal did not open.");
        modal.selectShowtime(SHOWTIME_INDEX);
        modal.confirmBooking();

        // 3. Seat selection — verify at least one seat is in booked (disabled) state
        SeatSelectionPage ssp = new SeatSelectionPage(driver);
        Assert.assertTrue(ssp.isBookedSeatDisabled(),
                "UI-09: Expected at least one booked (disabled) seat on seat selection page " +
                "after UI-08 booking was completed.");
    }

    // ── UI-10: Booking history ────────────────────────────────────────────────

    /**
     * UI-10: After UI-08, the booking history page should show >= 1 booking.
     */
    @Test(priority = 3,
          dependsOnMethods = "happyPath_UI08",
          description = "UI-10: Booking history should show at least 1 booking after happy-path")
    public void bookingHistory_UI10() {
        // 1. Login
        LoginPage lp = new LoginPage(driver).open();
        lp.enterEmail(BOOKER_EMAIL).enterPassword(BOOKER_PASSWORD).submit();
        Assert.assertTrue(lp.isLoggedIn(),
                "UI-10: Login failed for " + BOOKER_EMAIL);

        // 2. Open booking history
        BookingHistoryPage bhp = new BookingHistoryPage(driver).open();

        int count = bhp.count();
        Assert.assertTrue(count >= 1,
                "UI-10: Expected >= 1 booking in history page for " + BOOKER_EMAIL +
                " but found " + count);
    }

    // ── helpers ───────────────────────────────────────────────────────────────

    /** Parse an integer index from an Excel cell value; returns defaultVal on failure. */
    private int parseIndex(String value, int defaultVal) {
        try {
            if (value == null || value.trim().isEmpty()) return defaultVal;
            return Integer.parseInt(value.trim());
        } catch (NumberFormatException e) {
            return defaultVal;
        }
    }
}
