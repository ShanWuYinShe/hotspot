package com.hotspot.app.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Person
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.ui.components.ButtonSize
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.SegmentedTabs
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

/**
 * 登录 / 注册（移动 390 版式）：
 * - 用户名型 login_id 注册免验证码（手机号/邮箱型需验证码服务，当前未接入 → 只提示用户名注册）
 * - 登录态由 TokenStore StateFlow 驱动，成功后根导航自动切到主界面
 */
@Composable
fun LoginScreen(onLoggedIn: () -> Unit) {
    val colors = HotSpotTheme.colors
    val auth = AppContainer.authRepository
    val scope = rememberCoroutineScope()

    var isRegister by remember { mutableStateOf(false) }
    var loginId by remember { mutableStateOf("") }
    var nickname by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    fun submit() {
        if (loading) return
        if (loginId.isBlank() || password.isBlank() || (isRegister && nickname.isBlank())) {
            error = "请填写完整信息"
            return
        }
        scope.launch {
            loading = true
            error = null
            val result = if (isRegister) {
                auth.register(loginId.trim(), nickname.trim(), password)
            } else {
                auth.signin(loginId.trim(), password)
            }
            loading = false
            when (result) {
                is ApiResult.Ok -> onLoggedIn()
                is ApiResult.Err -> error = result.error.userMessage
            }
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = Spacing.xl),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Spacer(Modifier.height(Spacing.xxl))
        Icon(
            imageVector = Icons.Filled.Person,
            contentDescription = null,
            modifier = Modifier.size(Dimens.iconEmpty),
            tint = colors.accent,
        )
        Spacer(Modifier.height(Spacing.md))
        Text(
            text = "HotSpot",
            style = MaterialTheme.typography.displaySmall,
            fontWeight = FontWeight.SemiBold,
            color = colors.textPrimary,
        )
        Spacer(Modifier.height(Spacing.xs))
        Text(
            text = "在附近，遇见此刻发生的事",
            style = MaterialTheme.typography.bodyMedium,
            color = colors.textSecondary,
        )
        Spacer(Modifier.height(Spacing.xl))
        SegmentedTabs(
            options = listOf("登录", "注册"),
            selectedIndex = if (isRegister) 1 else 0,
            onSelected = { index ->
                isRegister = index == 1
                error = null
            },
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(Spacing.lg2))
        SystemInput(
            value = loginId,
            onValueChange = { loginId = it },
            placeholder = "手机号 / 邮箱 / 用户名",
            label = "登录标识",
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Text, imeAction = ImeAction.Next),
            modifier = Modifier.fillMaxWidth(),
        )
        if (isRegister) {
            Spacer(Modifier.height(Spacing.md))
            SystemInput(
                value = nickname,
                onValueChange = { nickname = it },
                placeholder = "展示给其他用户的昵称",
                label = "昵称",
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
                modifier = Modifier.fillMaxWidth(),
            )
        }
        Spacer(Modifier.height(Spacing.md))
        SystemInput(
            value = password,
            onValueChange = { password = it },
            placeholder = "密码",
            label = "密码",
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Password,
                imeAction = ImeAction.Done,
            ),
            keyboardActions = KeyboardActions(onDone = { submit() }),
            visualTransformation = PasswordVisualTransformation(),
            modifier = Modifier.fillMaxWidth(),
        )
        if (error != null) {
            Spacer(Modifier.height(Spacing.sm))
            Text(
                text = error ?: "",
                style = MaterialTheme.typography.bodySmall,
                color = colors.danger,
                modifier = Modifier.fillMaxWidth(),
            )
        }
        Spacer(Modifier.height(Spacing.lg))
        SystemButton(
            text = if (isRegister) "注册并登录" else "登录",
            onClick = { submit() },
            variant = ButtonVariant.PRIMARY,
            size = ButtonSize.LG,
            loading = loading,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(Spacing.md))
        Text(
            text = if (isRegister) {
                "注：首版仅支持用户名型注册（邮箱/手机号验证码服务尚未接入）"
            } else {
                "登录状态会保存在本机；退出登录在「我的 → 设置」"
            },
            style = MaterialTheme.typography.bodySmall,
            color = colors.textSecondary,
        )
        Spacer(Modifier.height(Spacing.xxl))
    }
}
