package platformapi

import "github.com/PastureStack/compose-cli/digest"

type Factory interface {
	Hash(service *PlatformService) (digest.ServiceHash, error)
	Create(service *PlatformService) error
	Upgrade(r *PlatformService, force bool, selected []string) error
	Rollback(r *PlatformService) error
}

func GetFactory(service *PlatformService) (Factory, error) {
	return &NormalFactory{}, nil
}
