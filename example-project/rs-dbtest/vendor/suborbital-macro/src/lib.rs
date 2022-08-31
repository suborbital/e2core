use proc_macro::TokenStream;
use quote::quote;
use syn::parse_macro_input;
use syn::DeriveInput;

#[proc_macro_derive(Runnable)]
pub fn derive_runnable(token_stream: TokenStream) -> TokenStream {
	let input = parse_macro_input!(token_stream as DeriveInput);

	let runnable_name = input.ident;

	let expanded = quote! {
		static RUNNABLE: &#runnable_name = &#runnable_name{};

		#[no_mangle]
		pub extern fn init() {
			suborbital::runnable::use_runnable(RUNNABLE);
		}
	};

	TokenStream::from(expanded)
}
