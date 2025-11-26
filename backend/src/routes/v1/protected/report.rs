use axum::{routing::post, Router};

use crate::{handlers::report, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/create", post(report::create))
        .route("/get_my_reports", post(report::get_my_reports))
        .route("/get_admin_list", post(report::get_admin_list))
        .route("/handle", post(report::handle))
}
