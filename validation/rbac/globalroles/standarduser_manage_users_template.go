package globalroles

import (
	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/tests/actions/rbac"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newCustomGlobalRole(verbs []string) v3.GlobalRole {
	name := namegen.AppendRandomString("custom-global-role-")
	return v3.GlobalRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{rbac.ManagementAPIGroup},
				Resources: []string{rbac.UsersResource},
				Verbs:     verbs,
			},
		},
	}
}

var (
	customGlobalRoleDelete      = newCustomGlobalRole([]string{"delete", "get", "list"})
	customGlobalRoleEdit        = newCustomGlobalRole([]string{"patch", "update", "get", "list"})
	customGlobalRoleManageUsers = newCustomGlobalRole([]string{rbac.ManageUsersVerb, "patch", "update", "delete", "get", "list"})
)
