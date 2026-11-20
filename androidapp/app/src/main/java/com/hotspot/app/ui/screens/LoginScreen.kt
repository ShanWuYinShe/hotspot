package com.hotspot.app.ui.screens

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import com.hotspot.app.AppContainer
import com.hotspot.app.R
import com.hotspot.app.data.api.ApiResult
import com.hotspot.app.ui.components.ButtonVariant
import com.hotspot.app.ui.components.SystemButton
import com.hotspot.app.ui.components.SystemInput
import com.hotspot.app.ui.components.hotSpotFocusRing
import com.hotspot.app.ui.theme.AppShapes
import com.hotspot.app.ui.theme.Dimens
import com.hotspot.app.ui.theme.HotSpotTextStyles
import com.hotspot.app.ui.theme.HotSpotTheme
import com.hotspot.app.ui.theme.Spacing
import kotlinx.coroutines.launch

/** 登录模式：首版登录/注册共用一张卡片、页内切换（无独立路由；忘记密码不在首版范围）。 */
private enum class AuthMode { LOGIN, REGISTER }

/** 表单动作：区分「提交登录/注册」与「进入访客模式」，只让当前动作展示 loading。 */
private enum class AuthAction { SUBMIT, GUEST }

// 设计登记几何（Figma 登录页 87:1818 / 表单卡 754:22）：卡片宽 = 屏宽 − 40、内边距 16、
// 输入 43/圆角 16、CTA 高 44。竖向节奏按设计稿行距近似取值（未登记为独立 token）。
private val LoginCardInset = 20.dp
private val LoginCardPadding = Spacing.lg
private val LoginCtaHeight = 44.dp
/** 字段组竖向间距 40（设计稿登录卡 754:22 中 account 输入底 → 密码标签顶的实测间距）。 */
private val LoginFieldGap = 40.dp

/** 品牌徽标底板：设计稿品牌行为黑色圆形徽标（品牌资产，不随主题变化，不属于语义色 token）。 */
private val LoginBrandPlate = Color(0xFF15171C)

/**
 * 登录 / 注册（390 移动版式，设计稿 87:1818）：
 * 氛围画布由根 MainActivity 统一挂载（design-system §16.3，页面不重复挂）→ 品牌行
 * → 表单卡片（eyebrow / 标题 / 副标题 / 账号 / 密码 / CTA / 协议 / 链接）→ 底部 slogan。
 * 登录态由 TokenStore StateFlow 驱动，成功后根导航自动切到主界面；访客模式失败（如 10400）
 * 透传服务端 message。
 */
@Composable
fun LoginScreen(onLoggedIn: () -> Unit) {
    val colors = HotSpotTheme.colors
    val auth = AppContainer.authRepository
    val scope = rememberCoroutineScope()

    var mode by remember { mutableStateOf(AuthMode.LOGIN) }
    var loginId by remember { mutableStateOf("") }
    var nickname by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var passwordVisible by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf<AuthAction?>(null) }
    var error by remember { mutableStateOf<String?>(null) }

    val isRegister = mode == AuthMode.REGISTER
    val idle = busy == null

    fun submit() {
        if (busy != null) return
        if (loginId.isBlank() || password.isBlank() || (isRegister && nickname.isBlank())) {
            error = "请填写完整信息"
            return
        }
        scope.launch {
            busy = AuthAction.SUBMIT
            error = null
            val result = if (isRegister) {
                auth.register(loginId.trim(), nickname.trim(), password)
            } else {
                auth.signin(loginId.trim(), password)
            }
            busy = null
            when (result) {
                is ApiResult.Ok -> onLoggedIn()
                is ApiResult.Err -> error = result.error.userMessage
            }
        }
    }

    fun continueAsGuest() {
        if (busy != null) return
        scope.launch {
            busy = AuthAction.GUEST
            error = null
            val result = auth.guest()
            busy = null
            when (result) {
                is ApiResult.Ok -> onLoggedIn()
                is ApiResult.Err -> error = result.error.userMessage
            }
        }
    }

    BoxWithConstraints(Modifier.fillMaxSize()) {
        val viewportMinHeight = maxHeight
        // 内容未超出视口时纵向居中（设计稿品牌行≈151/891）；超出时正常滚动。
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .heightIn(min = viewportMinHeight)
                .padding(horizontal = LoginCardInset, vertical = Spacing.xl),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            // 品牌行：32 圆形 logo + 18 semibold 品牌名 + 右侧 11 次级定位语
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(
                    modifier = Modifier
                        .size(Dimens.iconLg)
                        .clip(CircleShape)
                        .background(LoginBrandPlate),
                ) {
                    Image(
                        painter = painterResource(R.drawable.hotspot_logo),
                        contentDescription = null,
                        modifier = Modifier.fillMaxSize().clip(CircleShape),
                    )
                }
                Spacer(Modifier.width(Spacing.sm2))
                Text(
                    text = "HotSpot",
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.weight(1f))
                Text(
                    text = "城市社交画布",
                    style = HotSpotTextStyles.caption,
                    color = colors.textSecondary,
                )
            }
            Spacer(Modifier.height(Spacing.xxl))

            // 表单卡片：surface-primary + 圆角 20 + 轻投影 + 内边距 16
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .shadow(
                        elevation = 1.dp,
                        shape = AppShapes.card,
                        ambientColor = colors.shadowDark,
                        spotColor = colors.shadowDark,
                    )
                    .clip(AppShapes.card)
                    .background(colors.surface)
                    .padding(LoginCardPadding),
            ) {
                Text(
                    text = if (isRegister) "加入 HotSpot" else "欢迎回来",
                    style = HotSpotTextStyles.caption,
                    fontWeight = FontWeight.Medium,
                    color = colors.accent,
                )
                Spacer(Modifier.height(Spacing.sm))
                Text(
                    text = if (isRegister) "创建账号，探索附近。" else "回来看看附近的故事。",
                    style = MaterialTheme.typography.displaySmall,
                    fontWeight = FontWeight.SemiBold,
                    color = colors.textPrimary,
                )
                Spacer(Modifier.height(Spacing.sm))
                Text(
                    text = if (isRegister) {
                        "一个账号即可发现周边的人、群组与故事。"
                    } else {
                        "使用手机号、邮箱或账号登录。"
                    },
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.textSecondary,
                )
                Spacer(Modifier.height(Spacing.xxl))

                FieldLabel("手机号、邮箱或账号")
                Spacer(Modifier.height(Spacing.sm2))
                SystemInput(
                    value = loginId,
                    onValueChange = { loginId = it },
                    placeholder = "请输入账号",
                    leading = {
                        Icon(
                            painter = painterResource(R.drawable.ic_login_mail),
                            contentDescription = null,
                            modifier = Modifier.size(Dimens.iconXs),
                            tint = colors.textSecondary,
                        )
                    },
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Text,
                        imeAction = ImeAction.Next,
                    ),
                    enabled = idle,
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(LoginFieldGap))

                if (isRegister) {
                    FieldLabel("昵称")
                    Spacer(Modifier.height(Spacing.sm2))
                    SystemInput(
                        value = nickname,
                        onValueChange = { nickname = it },
                        placeholder = "展示给其他用户的昵称",
                        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
                        enabled = idle,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Spacer(Modifier.height(LoginFieldGap))
                }

                FieldLabel("密码")
                Spacer(Modifier.height(Spacing.sm2))
                SystemInput(
                    value = password,
                    onValueChange = { password = it },
                    placeholder = "请输入密码",
                    leading = {
                        Icon(
                            painter = painterResource(R.drawable.ic_login_lock),
                            contentDescription = null,
                            modifier = Modifier.size(Dimens.iconXs),
                            tint = colors.textSecondary,
                        )
                    },
                    trailing = {
                        // 44 触控热区（输入框 43 高时由父约束收窄到 43，视觉仍是图标）
                        Box(
                            modifier = Modifier
                                .size(Dimens.minTouchTarget)
                                .clip(CircleShape)
                                .clickable(
                                    interactionSource = remember { MutableInteractionSource() },
                                    indication = null,
                                ) { passwordVisible = !passwordVisible },
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(
                                painter = painterResource(
                                    if (passwordVisible) R.drawable.ic_visibility_off else R.drawable.ic_visibility,
                                ),
                                contentDescription = if (passwordVisible) "隐藏密码" else "显示密码",
                                modifier = Modifier.size(Dimens.iconXs),
                                tint = colors.textSecondary,
                            )
                        }
                    },
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Password,
                        imeAction = ImeAction.Done,
                    ),
                    keyboardActions = KeyboardActions(onDone = { submit() }),
                    visualTransformation = if (passwordVisible) {
                        VisualTransformation.None
                    } else {
                        PasswordVisualTransformation()
                    },
                    enabled = idle,
                    modifier = Modifier.fillMaxWidth(),
                )

                // 错误行常驻占位：空串 Text 高度会塌成 0，用 heightIn(min=15)（11/15 行高）
                // 预留一行，提示出现/消失时 CTA 不跳动（Web AuthForm 同策略）。
                Spacer(Modifier.height(Spacing.xxl))
                Text(
                    text = error.orEmpty(),
                    style = HotSpotTextStyles.caption,
                    color = colors.danger,
                    textAlign = TextAlign.Start,
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(min = 15.dp),
                )
                Spacer(Modifier.height(Spacing.lg2))

                SystemButton(
                    text = if (isRegister) "注册并登录" else "登录",
                    onClick = { submit() },
                    variant = ButtonVariant.PRIMARY,
                    height = LoginCtaHeight,
                    loading = busy == AuthAction.SUBMIT,
                    enabled = idle,
                    fillMaxWidth = true,
                    trailingIcon = {
                        Icon(
                            painter = painterResource(R.drawable.ic_arrow_forward),
                            contentDescription = null,
                            modifier = Modifier.size(Dimens.iconXs),
                            tint = colors.textOnAccent,
                        )
                    },
                )
                Spacer(Modifier.height(Spacing.xxl))

                // 协议行：无对应页面 → 两词用 accent 纯文本（不做假跳转）
                Text(
                    text = buildAnnotatedString {
                        append(if (isRegister) "注册即表示你同意 " else "登录即表示你同意 ")
                        withStyle(SpanStyle(color = colors.accent)) { append("服务条款") }
                        append(" 与 ")
                        withStyle(SpanStyle(color = colors.accent)) { append("隐私政策") }
                        append("。")
                    },
                    style = HotSpotTextStyles.caption,
                    color = colors.textSecondary,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(Spacing.lg))

                // 链接行 12 accent：注册态互相切换 + 访客模式（首版无忘记密码入口）
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.Center,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    if (isRegister) {
                        AuthLink(text = "已有账号？登录", enabled = idle) {
                            mode = AuthMode.LOGIN
                            error = null
                        }
                    } else {
                        AuthLink(text = "注册新账号", enabled = idle) {
                            mode = AuthMode.REGISTER
                            error = null
                        }
                    }
                    AuthLink(
                        text = if (busy == AuthAction.GUEST) "正在进入…" else "访客模式",
                        enabled = idle,
                        onClick = { continueAsGuest() },
                    )
                }
                if (isRegister) {
                    Spacer(Modifier.height(Spacing.sm))
                    Text(
                        text = "首版仅支持用户名注册（邮箱/手机号验证码服务尚未接入）。",
                        style = HotSpotTextStyles.caption,
                        color = colors.textSecondary,
                    )
                }
            }
            Spacer(Modifier.height(Spacing.xl))
            Text(
                text = "让附近有温度 · 随时可以退出",
                style = HotSpotTextStyles.caption,
                color = colors.textSecondary,
                textAlign = TextAlign.Center,
            )
        }
    }
}

/** 字段标签 11 次级（设计登记：登录卡标签 11）。 */
@Composable
private fun FieldLabel(text: String) {
    Text(
        text = text,
        style = HotSpotTextStyles.caption,
        color = HotSpotTheme.colors.textSecondary,
    )
}

/** 卡片内文字链接（12 accent，44 触控热区）。 */
@Composable
private fun AuthLink(
    text: String,
    enabled: Boolean = true,
    onClick: () -> Unit,
) {
    val colors = HotSpotTheme.colors
    Box(
        modifier = Modifier
            .heightIn(min = Dimens.minTouchTarget)
            .clip(AppShapes.iconTile)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                enabled = enabled,
                onClick = onClick,
            )
            .hotSpotFocusRing(cornerRadius = 12.dp)
            .padding(horizontal = Spacing.md2),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.bodyMedium,
            color = if (enabled) colors.accent else colors.textSecondary,
            maxLines = 1,
        )
    }
}
