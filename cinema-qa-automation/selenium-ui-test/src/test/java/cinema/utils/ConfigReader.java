package cinema.utils;

import java.io.FileInputStream;
import java.io.IOException;
import java.util.Properties;

public class ConfigReader {
    private static final Properties props = new Properties();
    static {
        try (FileInputStream fis = new FileInputStream("config/config.properties")) {
            props.load(fis);
        } catch (IOException e) {
            throw new RuntimeException("Failed to load config.properties", e);
        }
    }
    public static String get(String key) { return props.getProperty(key); }
    public static int getInt(String key) { return Integer.parseInt(props.getProperty(key)); }
    public static boolean getBool(String key) { return Boolean.parseBoolean(props.getProperty(key)); }
}
