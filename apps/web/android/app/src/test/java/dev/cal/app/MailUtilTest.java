package dev.cal.app;

import static org.junit.Assert.assertArrayEquals;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNull;

import org.junit.Test;

import java.util.Date;

public class MailUtilTest {

    @Test
    public void windowBounds_newestPage() {
        assertArrayEquals(new int[]{161, 200}, MailUtil.windowBounds(200, 0));
    }

    @Test
    public void windowBounds_olderPagesAndClamp() {
        assertArrayEquals(new int[]{121, 160}, MailUtil.windowBounds(200, 1));
        assertArrayEquals(new int[]{1, 37}, MailUtil.windowBounds(37, 0));
        assertArrayEquals(new int[]{1, 3}, MailUtil.windowBounds(123, 3));
    }

    @Test
    public void windowBounds_empty() {
        assertNull(MailUtil.windowBounds(0, 0));
        assertNull(MailUtil.windowBounds(200, 6));
    }

    @Test
    public void cleanHeader_stripsInjection() {
        assertEquals("hiBcc: evil@x", MailUtil.cleanHeader("hi\r\nBcc: evil@x"));
        assertEquals("", MailUtil.cleanHeader(null));
    }

    @Test
    public void iso_utcRfc3339() {
        assertEquals("2026-09-12T08:30:00Z", MailUtil.iso(new Date(1789201800000L)));
        assertEquals("", MailUtil.iso(null));
    }
}
