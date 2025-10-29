use axum::{routing::post, Router};

use crate::handlers::subscription::SubscriptionHandler;

pub fn subscription_routes() -> Router {
    Router::new().nest(
        "/protected/subscription",
        Router::new()
            .route("/get", post(SubscriptionHandler::get_subscriptions))
            .route("/update", post(SubscriptionHandler::update_subscription))
            .route("/update_batch", post(SubscriptionHandler::update_subscriptions_batch)),
    )
}

