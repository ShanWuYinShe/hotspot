pub mod v1;

use axum::{middleware, Router};

use crate::{
    middleware::auth::auth_middleware,
    state::AppState,
    websocket::routes::websocket_routes,
};

pub fn create_routes(state: AppState) -> Router {
    let public = v1::public_routes();
    let protected = v1::protected_routes()
        .layer(middleware::from_fn_with_state(
            state.clone(),
            auth_middleware,
        ));

    Router::new()
        .nest("/api/v1", public.merge(protected))
        .merge(websocket_routes())
        .with_state(state)
}
