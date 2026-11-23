package com.hotspot.app.util

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class ChatFileRulesTest {
    @Test
    fun allowsNormalDocument() {
        assertNull(ChatFileRules.blockedReason("report.pdf", 1024, "application/pdf"))
        assertNull(ChatFileRules.blockedReason("photo.jpg", 2048, "image/jpeg"))
        assertNull(ChatFileRules.blockedReason("archive.zip", 4096, "application/zip"))
    }

    @Test
    fun rejectsOversizeOrEmpty() {
        assertEquals("文件为空，无法发送", ChatFileRules.blockedReason("a.txt", 0, "text/plain"))
        assertEquals(
            "文件不能超过 100MB",
            ChatFileRules.blockedReason("big.bin", ChatFileRules.MAX_FILE_BYTES + 1, "application/octet-stream"),
        )
    }

    @Test
    fun rejectsBlockedExtensionsCaseInsensitively() {
        assertEquals("该文件类型不允许发送", ChatFileRules.blockedReason("setup.EXE", 100, "application/octet-stream"))
        assertEquals("该文件类型不允许发送", ChatFileRules.blockedReason("page.html", 100, "text/plain"))
        assertEquals("该文件类型不允许发送", ChatFileRules.blockedReason("icon.svg", 100, "image/svg+xml"))
    }

    @Test
    fun rejectsBlockedMimeEvenWithSafeExtension() {
        assertEquals("该文件类型不允许发送", ChatFileRules.blockedReason("fake.txt", 100, "text/html; charset=utf-8"))
        assertEquals("该文件类型不允许发送", ChatFileRules.blockedReason("fake.txt", 100, "application/javascript"))
    }
}
