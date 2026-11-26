package com.hotspot.app.data.update

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class UpdateCheckerTest {
    @Test fun newerPatchWins() {
        assertTrue(isNewerVersion("1.15.35", "1.15.34"))
    }

    @Test fun equalIsNotNewer() {
        assertFalse(isNewerVersion("1.15.35", "1.15.35"))
    }

    @Test fun higherMinorBeatsPatch() {
        assertTrue(isNewerVersion("1.16.0", "1.15.99"))
    }

    @Test fun vPrefixAndShortVersion() {
        assertTrue(isNewerVersion("v2", "1.99.99"))
    }
}
