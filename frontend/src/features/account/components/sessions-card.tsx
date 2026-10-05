import { LogOutIcon } from "lucide-react"
import { useNavigate } from "react-router"
import { toast } from "sonner"

import { ConfirmDialog } from "@/components/common/confirm-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { failed } from "@/lib/notify"
import { useLogoutAll } from "@/lib/queries"

/** Ends all sessions of the user, e.g. after using a shared computer. */
export function SessionsCard() {
  const navigate = useNavigate()
  const logoutAll = useLogoutAll({
    onSuccess: () => {
      toast.success("Signed out on all devices")
      navigate("/login", { replace: true })
    },
    onError: failed("sign out"),
  })
  return (
    <Card>
      <CardHeader>
        <CardTitle>Sessions</CardTitle>
        <CardDescription>
          Signs you out in every browser, including this one. API tokens keep working; revoke them below if needed.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ConfirmDialog
          trigger={
            <Button variant="outline">
              <LogOutIcon />
              Sign out everywhere
            </Button>
          }
          title="Sign out on all devices?"
          description="All sessions end immediately. You have to sign in again here, too."
          confirmLabel="Sign out everywhere"
          onConfirm={() => logoutAll.mutate()}
        />
      </CardContent>
    </Card>
  )
}
