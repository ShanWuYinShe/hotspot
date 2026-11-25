package com.hotspot.app.data.media

import java.io.FileNotFoundException
import java.io.IOException
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class MediaReadErrorsTest {
    @Test
    fun fileNotFoundBecomesOfflinePhotoMessage() {
        assertEquals(
            "照片尚未下载到本机，请换一个或先在相册中下载",
            MediaReadErrors.describe(FileNotFoundException(), "照片"),
        )
    }

    @Test
    fun securityExceptionBecomesPermissionMessage() {
        assertEquals(
            "没有访问该视频的权限，请重新选择",
            MediaReadErrors.describe(SecurityException(), "视频"),
        )
    }

    @Test
    fun otherErrorsKeepOriginalMessage() {
        assertNull(MediaReadErrors.describe(IOException("disk"), "文件"))
    }
}
