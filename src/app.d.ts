declare global {
	namespace App {
		interface Error {
			message: string;
			stack?: string;
		}
		interface Locals {
			user: { username: string; name: string; isAdmin: boolean } | null;
		}
	}
}

export {};
