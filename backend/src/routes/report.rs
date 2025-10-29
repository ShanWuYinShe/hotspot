use axum::{routing::post, Router};

use crate::handlers::report::ReportHandler;

pub fn report_routes() -> Router {
    Router::new().nest(
        "/protected/report",
        Router::new()
            .route("/create", post(ReportHandler::create_report))
            .route("/history", post(ReportHandler::get_report_history))
            .route("/all", post(ReportHandler::get_all_reports))
            .route("/review", post(ReportHandler::review_report)),
    )
}

