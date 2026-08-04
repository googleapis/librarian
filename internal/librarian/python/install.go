// limitations under the License.

package python

import (
  "context"

  "github.com/googleapis/librarian/internal/config"
  "github.com/googleapis/librarian/internal/tool/pip"
)

// Install installs Python pip tool dependencies.
func Install(ctx context.Context, tools *config.Tools) error {
  if tools == nil || len(tools.Pip) == 0 {
    return nil
  }
  return pip.Install(ctx, tools.Pip)
}
