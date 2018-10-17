package backend

func initAdmin() {
	Server.GET("/admin", AdminHandle(func(c ctx, user *User) error {
		return MFI.ServeFile(c, "/admin.html")
	}))
}
