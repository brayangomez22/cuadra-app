// Command probe imports a module, which the composition rule forbids.
package main

import sales "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"

func main() { _ = sales.Total }
