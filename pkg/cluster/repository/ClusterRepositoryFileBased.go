package repository

import (
	"encoding/json"
	"errors"
	"github.com/devtron-labs/devtron/pkg/sql"
	"os"
	"path"

	//"database/sql"
	//"encoding/json"
	"github.com/glebarez/sqlite"
	"github.com/go-pg/pg"
	//_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ClusterRepositoryFileBased struct {
	//*ClusterRepositoryImpl
	logger       *zap.SugaredLogger
	dbConnection *gorm.DB
}

type ClusterEntity struct {
	ID                int
	ClusterName       string
	ServerUrl         string
	Active            *bool
	Config            string
	K8sVersion        string
	ErrorInConnecting string
	sql.AuditLog
}

func NewClusterRepositoryFileBased(logger *zap.SugaredLogger) *ClusterRepositoryFileBased {
	//clusterRepositoryImpl := NewClusterRepositoryImpl(nil, logger)
	err, clusterDbPath := createOrCheckClusterDbPath(logger)
	db, err := gorm.Open(sqlite.Open(clusterDbPath), &gorm.Config{})
	//db, err := sql.Open("sqlite3", "./cluster.db")
	if err != nil {
		logger.Fatal("error occurred while opening db connection", "error", err)
	}
	migrator := db.Migrator()
	clusterEntity := &ClusterEntity{}
	hasTable := migrator.HasTable(clusterEntity)
	if !hasTable {
		err = migrator.CreateTable(clusterEntity)
		if err != nil {
			logger.Fatal("error occurred while creating cluster table", "error", err)
		}
	}
	logger.Info("cluster repository file based initialized")
	return &ClusterRepositoryFileBased{logger, db}
}

func createOrCheckClusterDbPath(logger *zap.SugaredLogger) (error, string) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Errorw("error occurred while finding home dir", "err", err)
		return err, ""
	}
	clusterDbDir := path.Join(userHomeDir, "./.kube/.devtron")
	err = os.MkdirAll(clusterDbDir, os.ModePerm)
	if err != nil {
		logger.Errorw("error occurred while creating db", "path", clusterDbDir, "err", err)
		return err, ""
	}

	clusterDbPath := path.Join(clusterDbDir, "./cluster.db")
	return nil, clusterDbPath
}

func (impl *ClusterRepositoryFileBased) Save(model *Cluster) error {

	err, clusterEntity := impl.convertToEntity(model)
	if err != nil {
		return err
	}
	result := impl.dbConnection.Model(clusterEntity).Create(clusterEntity)
	err = result.Error

	//query := "INSERT INTO cluster (cluster_name, server_url, active, config) VALUES(?, ?, ?, ?)"
	//stmt, err := impl.dbConnection.Prepare(query)
	//if err != nil {
	//	impl.logger.Errorw("error occurred while preparing statement ", "query", query, "error", err)
	//	return err
	//}
	//configJson, err := json.Marshal(model.Config)
	//result, err := stmt.Exec(model.ClusterName, model.ServerUrl, 't', configJson)
	//defer stmt.Close()
	if err != nil {
		impl.logger.Errorw("error occurred while executing insert statement", "err", err)
		return err
	}
	//lastInsertedId, _ := result.LastInsertId()
	model.Id = clusterEntity.ID
	return nil
}

func (impl *ClusterRepositoryFileBased) convertToEntity(model *Cluster) (error, *ClusterEntity) {
	configJson, err := json.Marshal(model.Config)
	if err != nil {
		impl.logger.Errorw("error occurred while converting to entity", "model", model, "err", err)
		return errors.New("failed to process cluster data"), nil
	}
	clusterEntity := &ClusterEntity{
		ID:                model.Id,
		ClusterName:       model.ClusterName,
		ServerUrl:         model.ServerUrl,
		Config:            string(configJson),
		Active:            &model.Active,
		K8sVersion:        model.K8sVersion,
		ErrorInConnecting: model.ErrorInConnecting,
		AuditLog:          sql.AuditLog{UpdatedOn: model.UpdatedOn, CreatedOn: model.CreatedOn, UpdatedBy: model.UpdatedBy, CreatedBy: model.CreatedBy},
	}
	return err, clusterEntity
}

func (impl *ClusterRepositoryFileBased) convertToModel(entity *ClusterEntity) (*Cluster, error) {
	clusterConfig := make(map[string]string)
	if len(entity.Config) > 0 {
		err := json.Unmarshal([]byte(entity.Config), &clusterConfig)
		if err != nil {
			impl.logger.Errorw("error occured while unmarshalling cluster config ", "error", err)
			return nil, errors.New("failed to process cluster data")
		}
	}
	isActive := false
	if entity.Active != nil {
		isActive = *entity.Active
	}
	clusterBean := &Cluster{
		Id:                entity.ID,
		ClusterName:       entity.ClusterName,
		ServerUrl:         entity.ServerUrl,
		Config:            clusterConfig,
		K8sVersion:        entity.K8sVersion,
		ErrorInConnecting: entity.ErrorInConnecting,
		AuditLog:          entity.AuditLog,
		Active:            isActive,
	}
	return clusterBean, nil
}

func (impl *ClusterRepositoryFileBased) FindOne(clusterName string) (*Cluster, error) {
	return impl.FindOneActive(clusterName)
}

func (impl *ClusterRepositoryFileBased) FindOneActive(clusterName string) (*Cluster, error) {
	clusterEntity := &ClusterEntity{}
	result := impl.dbConnection.
		Where("ClusterName = ?", clusterName).
		Where("Active = ?", true).
		Find(clusterEntity).
		Limit(1)
	err := result.Error
	if err != nil {
		impl.logger.Errorw("error occurred while finding cluster data ", "clusterName", clusterName, "err", err)
		return nil, err
	}
	//queryRow := impl.dbConnection.QueryRow("SELECT * FROM clusterEntity WHERE cluster_name = ? and active = ?")
	//err := queryRow.Scan()
	clusterBean, err := impl.convertToModel(clusterEntity)
	if err != nil {
		impl.logger.Errorw("error occurred while converting cluster data to  model ", "clusterName", clusterName, "err", err)
		return nil, errors.New("failed to fetch cluster")
	}
	return clusterBean, nil
}

func (impl *ClusterRepositoryFileBased) FindAll() ([]Cluster, error) {
	return impl.FindAllActive()
}

func (impl *ClusterRepositoryFileBased) FindAllActive() ([]Cluster, error) {
	var clusterEntities []ClusterEntity
	result := impl.dbConnection.
		Where("Active = ?", true).
		Find(&clusterEntities)
	err := result.Error
	if err != nil {
		impl.logger.Errorw("error occurred while finding all cluster data", "err", err)
		return nil, err
	}
	var clusters []Cluster
	for _, clusterEntity := range clusterEntities {
		clusterBean, err := impl.convertToModel(&clusterEntity)
		if err != nil {
			impl.logger.Errorw("error occurred while converting entity to model bean", "err", err)
			continue
		}
		clusters = append(clusters, *clusterBean)
	}
	return clusters, nil
}

func (impl *ClusterRepositoryFileBased) FindById(id int) (*Cluster, error) {
	clusterEntity := &ClusterEntity{}
	result := impl.dbConnection.
		Where("ID =?", id).
		Where("Active = ?", true).
		Find(clusterEntity).
		Limit(1)
	err := result.Error
	if err != nil {
		impl.logger.Errorw("error occurred while finding cluster data ", "id", id, "err", err)
		return nil, err
	}
	//queryRow := impl.dbConnection.QueryRow("SELECT * FROM clusterEntity WHERE cluster_name = ? and active = ?")
	//err := queryRow.Scan()
	clusterBean, err := impl.convertToModel(clusterEntity)
	if err != nil {
		impl.logger.Errorw("error occurred while converting cluster data to  model ", "id", id, "err", err)
		return nil, errors.New("failed to fetch cluster")
	}
	return clusterBean, nil
}

func (impl *ClusterRepositoryFileBased) FindByIds(id []int) ([]Cluster, error) {

	var clusterEntities []ClusterEntity
	result := impl.dbConnection.
		Where("ID in(?)", pg.In(id)).
		Where("Active = ?", true).
		Find(&clusterEntities)
	err := result.Error
	if err != nil {
		impl.logger.Errorw("error occurred while finding all cluster data", "err", err)
		return nil, err
	}
	var clusters []Cluster
	for _, clusterEntity := range clusterEntities {
		clusterBean, err := impl.convertToModel(&clusterEntity)
		if err != nil {
			impl.logger.Errorw("error occurred while converting entity to model bean", "err", err)
			continue
		}
		clusters = append(clusters, *clusterBean)
	}
	return clusters, nil
	//var cluster []Cluster
	//result := impl.dbConnection.
	//	Find(&cluster).
	//	Where("id in(?)", pg.In(id)).
	//	Where("active =?", true)
	//return cluster, result.Error
}

func (impl *ClusterRepositoryFileBased) Update(model *Cluster) error {
	err, entity := impl.convertToEntity(model)
	if err != nil {
		impl.logger.Errorw("error occurred while converting model to entity", "model", model, "error", err)
		return errors.New("failed to update cluster")
	}
	result := impl.dbConnection.Model(entity).Updates(entity)
	err = result.Error
	if err != nil {
		impl.logger.Errorw("error occurred while updating cluster", "model", model, "error", err)
		return errors.New("failed to update cluster")
	}
	return nil
	//return impl.dbConnection.Update(model)
}

func (impl *ClusterRepositoryFileBased) Delete(model *Cluster) error {
	err, entity := impl.convertToEntity(model)
	if err != nil {
		impl.logger.Errorw("error occurred while converting model to entity", "model", model, "error", err)
		return errors.New("failed to delete cluster")
	}
	result := impl.dbConnection.Delete(entity)
	err = result.Error
	if err != nil {
		impl.logger.Errorw("error occurred while deleting cluster", "model", model, "err", err)
		return errors.New("failed to delete cluster")
	}
	return nil
}

func (impl *ClusterRepositoryFileBased) MarkClusterDeleted(model *Cluster) error {
	model.Active = false
	return impl.Update(model)
}

func (impl *ClusterRepositoryFileBased) UpdateClusterConnectionStatus(clusterId int, errorInConnecting string) error {

	result := impl.dbConnection.Model(&ClusterEntity{}).Where("ID = ?", clusterId).Update("ErrorInConnecting = ?", errorInConnecting)
	err := result.Error
	if err != nil {
		impl.logger.Errorw("error occurred while updating cluster connection status", "clusterId", clusterId, "error", errorInConnecting, "err", err)
		return errors.New("failed to update cluster status")
	}
	return nil
}
