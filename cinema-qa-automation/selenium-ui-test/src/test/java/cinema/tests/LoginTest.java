package cinema.tests;

import cinema.base.BaseTest;
import cinema.pages.LoginPage;
import cinema.utils.ConfigReader;
import cinema.utils.DBHelper;
import cinema.utils.ExcelReader;
import org.testng.Assert;
import org.testng.annotations.BeforeClass;
import org.testng.annotations.DataProvider;
import org.testng.annotations.Test;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.Map;

/**
 * UI-04, UI-05 — Login tests.
 *
 * LoginData sheet columns:
 *   tcId | email | password | expectedResult | expectedMessage
 *
 * UI-04: valid credentials → isLoggedIn() == true
 *        (Note: if email verification is required by the BE, the test may see
 *         a "chưa xác thực" / "verify" error instead of a successful login.
 *         That is a known limitation and is accepted behaviour per spec.)
 * UI-05: wrong password → getErrorMessage() contains expectedMessage (case-insensitive)
 */
public class LoginTest extends BaseTest {

    private static final String DATA_FILE = "test-data/ClientFlowData.xlsx";
    private static final String SEED_EMAIL    = "qa_test_login@example.com";
    private static final String SEED_PASSWORD = "Pass@1234";

    /**
     * Pre-condition: ensure qa_test_login@example.com is registered before running login tests.
     * Registers the user via API if they are not yet in the DB.
     * Network/API errors are logged but do NOT abort the test suite.
     */
    @BeforeClass(alwaysRun = true, dependsOnMethods = "setUp")
    public void seedLoginUser() {
        try {
            if (DBHelper.userExists(SEED_EMAIL)) {
                System.out.println("[Seed] Login user already in DB, skipping.");
                return;
            }
        } catch (Exception e) {
            System.err.println("[Seed] DB check failed (DB may be offline): " + e.getMessage());
        }
        try {
            HttpClient client = HttpClient.newHttpClient();
            String body = String.format(
                "{\"email\":\"%s\",\"password\":\"%s\",\"confirmPassword\":\"%s\"," +
                "\"firstName\":\"Login\",\"lastName\":\"Tester\",\"address\":\"456 Login Ave\"}",
                SEED_EMAIL, SEED_PASSWORD, SEED_PASSWORD);
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(ConfigReader.get("api.base.url") + "/api/v1/auth/register"))
                    .header("Content-Type", "application/json")
                    .POST(HttpRequest.BodyPublishers.ofString(body))
                    .build();
            HttpResponse<String> response =
                    client.send(request, HttpResponse.BodyHandlers.ofString());
            System.out.println("[Seed] Seeded login user via API. Status: " + response.statusCode()
                    + " Body: " + response.body());
        } catch (Exception e) {
            System.err.println("[Seed] Warning — could not seed login user: " + e.getMessage());
        }
    }

    @DataProvider(name = "loginData")
    public Object[][] loginData() {
        return ExcelReader.toDataProvider(DATA_FILE, "LoginData");
    }

    /**
     * Parameterised login test driven by LoginData sheet.
     *
     * For PASS: assert isLoggedIn() is true.
     *           If the app requires email verification, the test will likely see a
     *           verification-required message and isLoggedIn() will be false.
     *           A console warning is printed in that case but the assertion is still enforced.
     *
     * For FAIL: assert getErrorMessage() contains expectedMessage (case-insensitive).
     */
    @Test(dataProvider = "loginData",
          description = "Login with various credentials — UI-04, UI-05")
    public void login(Map<String, String> row) {
        String email           = row.get("email");
        String password        = row.get("password");
        String expectedResult  = row.get("expectedResult");
        String expectedMessage = row.get("expectedMessage");

        LoginPage lp = new LoginPage(driver).open();
        lp.enterEmail(email)
          .enterPassword(password)
          .submit();

        if ("PASS".equalsIgnoreCase(expectedResult)) {
            boolean loggedIn = lp.isLoggedIn();
            if (!loggedIn) {
                String errMsg = lp.getErrorMessage();
                System.err.println("[" + row.get("tcId") + "] Login failed. Error message: " + errMsg);
                System.err.println("  Hint: User may require email verification via DB/admin.");
            }
            Assert.assertTrue(loggedIn,
                    "[" + row.get("tcId") + "] Login expected to succeed for " + email);
        } else {
            String err      = lp.getErrorMessage().toLowerCase();
            String expected = (expectedMessage != null ? expectedMessage : "").toLowerCase();
            Assert.assertTrue(
                    expected.isEmpty() || err.contains(expected),
                    "[" + row.get("tcId") + "] Expected error to contain: \"" + expected +
                    "\" but got: \"" + err + "\"");
        }
    }
}
