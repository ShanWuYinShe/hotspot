#[macro_export]
macro_rules! feed_dismiss_clause_moment_required {
    ($user_param:literal, $alias:ident) => {
        concat!(
            " AND NOT EXISTS (\n",
            "     SELECT 1 FROM feed_dismissals fd\n",
            "     WHERE fd.user_id = $",
            stringify!($user_param),
            "\n",
            "       AND fd.target_type = 'moment'\n",
            "       AND fd.target_id = ",
            stringify!($alias),
            ".moment_id::text\n",
            " )\n",
            " AND NOT EXISTS (\n",
            "     SELECT 1 FROM feed_dismissals fd\n",
            "     WHERE fd.user_id = $",
            stringify!($user_param),
            "\n",
            "       AND fd.target_type = 'user'\n",
            "       AND fd.target_id = ",
            stringify!($alias),
            ".user_id::text\n",
            " )\n"
        )
    };
}

#[macro_export]
macro_rules! feed_dismiss_clause_moment_optional {
    ($user_param:literal, $alias:ident) => {
        concat!(
            " AND (\n",
            "     $",
            stringify!($user_param),
            "::bigint IS NULL\n",
            "     OR (\n",
            "         NOT EXISTS (\n",
            "             SELECT 1 FROM feed_dismissals fd\n",
            "             WHERE fd.user_id = $",
            stringify!($user_param),
            "\n",
            "               AND fd.target_type = 'moment'\n",
            "               AND fd.target_id = ",
            stringify!($alias),
            ".moment_id::text\n",
            "         )\n",
            "         AND NOT EXISTS (\n",
            "             SELECT 1 FROM feed_dismissals fd\n",
            "             WHERE fd.user_id = $",
            stringify!($user_param),
            "\n",
            "               AND fd.target_type = 'user'\n",
            "               AND fd.target_id = ",
            stringify!($alias),
            ".user_id::text\n",
            "         )\n",
            "     )\n",
            " )\n"
        )
    };
}

#[macro_export]
macro_rules! feed_dismiss_clause_single_optional {
    ($user_param:literal, $alias:ident, $target_type:literal, $id_expr:expr) => {
        concat!(
            " AND (\n",
            "     $",
            stringify!($user_param),
            "::bigint IS NULL\n",
            "     OR NOT EXISTS (\n",
            "         SELECT 1 FROM feed_dismissals fd\n",
            "         WHERE fd.user_id = $",
            stringify!($user_param),
            "\n",
            "           AND fd.target_type = '",
            $target_type,
            "'\n",
            "           AND fd.target_id = ",
            $id_expr,
            "\n",
            "     )\n",
            " )\n"
        )
    };
}
