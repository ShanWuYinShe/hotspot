use std::fs;
use std::path::{Path, PathBuf};

fn backend_src_dir() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR")).join("src")
}

fn read_source(relative_path: &str) -> String {
    let path = backend_src_dir().join(relative_path);
    fs::read_to_string(&path)
        .unwrap_or_else(|error| panic!("failed to read {}: {}", path.display(), error))
}

fn extract_function_body(source: &str, fn_name: &str) -> String {
    let signature = format!("fn {}", fn_name);
    let start = source
        .find(&signature)
        .unwrap_or_else(|| panic!("function {} not found", fn_name));
    let body_start = source[start..]
        .find('{')
        .map(|index| start + index)
        .unwrap_or_else(|| panic!("function {} body start not found", fn_name));

    let mut depth = 0usize;
    let mut end = None;
    for (index, ch) in source[body_start..].char_indices() {
        match ch {
            '{' => depth += 1,
            '}' => {
                depth -= 1;
                if depth == 0 {
                    end = Some(body_start + index + 1);
                    break;
                }
            }
            _ => {}
        }
    }

    let end = end.unwrap_or_else(|| panic!("function {} body end not found", fn_name));
    source[start..end].to_string()
}

fn assert_function_contains(relative_path: &str, fn_name: &str, expected_snippets: &[&str]) {
    let source = read_source(relative_path);
    let body = extract_function_body(&source, fn_name);
    for snippet in expected_snippets {
        assert!(
            body.contains(snippet),
            "expected function {} in {} to contain snippet: {}\nfunction body:\n{}",
            fn_name,
            relative_path,
            snippet,
            body
        );
    }
}

#[test]
fn register_must_record_user_in_bloom() {
    assert_function_contains(
        "services/auth_service.rs",
        "register",
        &[".create(", ".record_user(user.user_id)"],
    );
}

#[test]
fn guest_creation_must_record_user_in_bloom() {
    assert_function_contains(
        "services/auth_service.rs",
        "create_guest",
        &[".create(", ".record_user(user.user_id)"],
    );
}

#[test]
fn super_admin_bootstrap_must_record_user_in_bloom() {
    assert_function_contains(
        "services/admin_rbac_service.rs",
        "ensure_super_admin_bootstrap",
        &[".create(", ".record_user(user.user_id)"],
    );
}

#[test]
fn group_creation_must_record_group_in_bloom() {
    assert_function_contains(
        "services/group_service/crud.rs",
        "create_group",
        &[".create(", ".record_group(&group.group_id.to_string())"],
    );
}

#[test]
fn moment_creation_must_record_moment_in_bloom() {
    assert_function_contains(
        "services/moment_service.rs",
        "create_moment",
        &[".create(", ".record_moment(&moment.moment_id.to_string())"],
    );
}

#[test]
fn comment_creation_must_record_comment_in_bloom() {
    assert_function_contains(
        "services/comment_service.rs",
        "add_comment",
        &[".create(", ".record_comment(&row.comment_id.to_string())"],
    );
}

#[test]
fn group_message_send_must_record_message_in_bloom() {
    assert_function_contains(
        "services/group_message_service.rs",
        "send",
        &[".create(", ".record_message(&created.message_id.to_string())"],
    );
}

#[test]
fn private_message_send_must_record_message_in_bloom() {
    assert_function_contains(
        "services/private_message_service.rs",
        "send",
        &[".create(", ".record_message(&created.message_id.to_string())"],
    );
}