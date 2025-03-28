package repositories

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"AgroXpert-Backend/src/database"
	"AgroXpert-Backend/src/models"
)

func GetAllHarvests() ([]models.Harvest, error) {
	var resultHarvest []models.Harvest
	var modelHarvest models.Harvest
	collection := database.Db.GetCollection("Harvest")
	filter := bson.M{}

	harvest, err := collection.Find(context.Background(), filter)
	if err != nil {
		return nil, fmt.Errorf("error fiend all harvest: %v", err)
	}

	for harvest.Next(context.Background()) {
		err := harvest.Decode(&modelHarvest)
		if err != nil {
			return nil, fmt.Errorf("error decode harvest: %v", err)
		}

		resultHarvest = append(resultHarvest, modelHarvest)
	}

	return resultHarvest, nil
}

func GetOneHarvest(HarvestID string) (models.Harvest, error) {
	var modelHarvest models.Harvest
	collection := database.Db.GetCollection("Harvest")

	id, err := primitive.ObjectIDFromHex(HarvestID)
	if err != nil {
		return models.Harvest{}, fmt.Errorf("error convert id: %v", err)
	}

	filter := bson.M{"_id": id}
	harvest := collection.FindOne(context.Background(), filter)
	err = harvest.Decode(&modelHarvest)
	if err == mongo.ErrNoDocuments {
		return models.Harvest{}, err
	}

	if err != nil {
		return models.Harvest{}, fmt.Errorf("error decode farm lot: %v", err)
	}

	return modelHarvest, nil
}
func GetHarvestsByFarmLotID(FarmLotID string) ([]models.Harvest, error) {
	collection := database.Db.GetCollection("Harvest")
	id, err := primitive.ObjectIDFromHex(FarmLotID)
	if err != nil {
		return nil, fmt.Errorf("error converting ID: %v", err)
	}
	filter := bson.M{"idFarmLot": id}
	resultHarvest, err := fetchHarvests(collection, filter)
	if err != nil {
		return nil, err
	}
	return resultHarvest, nil
}

func fetchHarvests(collection *mongo.Collection, filter bson.M) ([]models.Harvest, error) {
	var resultHarvest []models.Harvest
	modelHarvest := models.Harvest{}
	harvestCursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return nil, fmt.Errorf("error finding harvests: %v", err)
	}
	defer harvestCursor.Close(context.Background())

	for harvestCursor.Next(context.Background()) {
		err := harvestCursor.Decode(&modelHarvest)
		if err != nil {
			return nil, fmt.Errorf("error decoding harvest: %v", err)
		}
		resultHarvest = append(resultHarvest, modelHarvest)
	}

	return resultHarvest, nil
}
func CreateHarvest(harvestReq models.CreateHarvest) (models.CreateHarvest, error) {
	collection := database.Db.GetCollection("Harvest")
	harvest := createHarvestMap(harvestReq)

	result, err := insertDocument(collection, harvest)
	if err != nil {
		return models.CreateHarvest{}, fmt.Errorf("error inserting farm lot: %v", err)
	}

	id := result.InsertedID.(primitive.ObjectID)
	harvestReq.ID = id
	return harvestReq, nil
}

func createHarvestMap(harvestReq models.CreateHarvest) bson.M {
	harvest := bson.M{
		"type":                   harvestReq.Type,
		"idFarmLot":              harvestReq.IDFarmLot,
		"evaluationStartDate":    harvestReq.EvaluationStartDate + "Z",
		"evaluationEndDate":      harvestReq.EvaluationEndDate + "Z",
		"summaryFinalProduction": nil,
		"estimates":              []primitive.ObjectID{},
	}

	return harvest
}

func insertDocument(collection *mongo.Collection, document interface{}) (*mongo.InsertOneResult, error) {
	result, err := collection.InsertOne(context.Background(), document)
	if err != nil {
		return nil, err
	}
	return result, nil
} 

func UpdateSummaryFinalProduction(idHarvest string, idFinalProduction primitive.ObjectID) error {
	collection := database.Db.GetCollection("Harvest")

	idHarvestUpdate, err := primitive.ObjectIDFromHex(idHarvest)
	if err != nil {
		return fmt.Errorf("error convert id: %v", err)
	}

	filter := bson.M{"_id": idHarvestUpdate}
	update := bson.M{"$set": bson.M{"summaryFinalProduction": idFinalProduction}}
	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return fmt.Errorf("error update summary final production: %v", err)
	}

	return nil
}


// Actualiza la lista de las estimaciones de cosechas asociadas a una cosecha
// específica. Esta función es necesaria para mantener un registro preciso y actualizado de las estimaciones 
//realizadas a lo largo del tiempo. Al agregar nuevas, se permite un seguimiento 
//más efectivo y una evaluación del rendimiento y las expectativas de la cosecha en cuestión.

// @param idHarvest: Es el id de la cosecha a la que se desea agregar la nueva estimación. Con este se
// buscará y actualizará la cosecha correspondiente en la base de datos.

// @param idNewEstimate: Es el id de la nueva estimación que se desea agregar a la lista.

// @return error: Un error que indica un fallo en la operación. Si no hay errores, se devuelve 'nil'.

func UpdateEstimatesHarvest(idHarvest string, idNewEstimate primitive.ObjectID) error {
	collection := database.Db.GetCollection("Harvest")

	idHarvestUpdate, err := primitive.ObjectIDFromHex(idHarvest)
	if err != nil {
		return fmt.Errorf("error convert id: %v", err)
	}

	filter := bson.M{"_id": idHarvestUpdate}
	update := bson.M{
		"$push": bson.M{"estimates": idNewEstimate},
	}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return fmt.Errorf("error update estimates harvest : %v ", err)
	}

	return nil
}

// Recupera y consolida la información histórica de las estimaciones de cosechas
// y la producción final para un lote de granja específico. Este método es útil para analizar tendencias
// y realizar un seguimiento del rendimiento de la producción agrícola a lo largo del tiempo.

// @param FarmLotID: Es el identificador único del lote de granja para el cual se desea recuperar la
//   información histórica de las estimaciones de cosechas.

// @return []models.HarvestDetails: La información histórica de las estimaciones de cosecha y producción 
//  final para el lote de granja especificado. Si no se encuentran resultados, se devuelve un slice vacío

// @return error: Un error que indica un fallo en la operación. Si no hay errores, se devuelve 'nil'.
func GetHistoricHarvestEsimation(FarmLotID string) ([]models.HarvestDetails, error) {
	collection := database.Db.GetCollection("Harvest")
	id, err := primitive.ObjectIDFromHex(FarmLotID)
	pipelineHistoric := buildPipelineHistoric(id)
	resultHarvest, err := fetchHistoricHarvests(collection, pipelineHistoric)
	if err != nil {
		return nil, err
	}
	return resultHarvest, nil
}

func buildPipelineHistoric(id primitive.ObjectID) []bson.M {
	pipelineHistoric := []bson.M{
		{
			"$match": bson.M{
				"idFarmLot": id,
			},
		},
		{
			"$lookup": bson.M{
				"from": "Estimates",
				"let":  bson.M{"idsEstimates": "$estimates"},
				"pipeline": []bson.M{
					{"$match": bson.M{
						"$expr": bson.M{
							"$in": []string{"$_id", "$$idsEstimates"},
						},
					},
					},
				},
				"as": "estimates",
			},
		},
		{
			"$lookup": bson.M{
				"from":         "FinalProduction",
				"localField":   "summaryFinalProduction",
				"foreignField": "_id",
				"as":           "summaryFinalProduction",
			},
		},
		{
			"$unwind": bson.M{
				"path":                       "$summaryFinalProduction",
				"preserveNullAndEmptyArrays": true,
			},
		},
		{
			"$addFields": bson.M{
				"summaryFinalProduction": bson.M{"$ifNull": bson.A{"$summaryFinalProduction", nil}},
			},
		},
	}
	return pipelineHistoric
}

func fetchHistoricHarvests(collection *mongo.Collection, pipeline []bson.M) ([]models.HarvestDetails, error) {
	var resultHarvest []models.HarvestDetails
	modelHarvestDetails := models.HarvestDetails{}

	historicCursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, fmt.Errorf("error aggregating historic harvests: %v", err)
	}
	defer historicCursor.Close(context.Background())

	for historicCursor.Next(context.Background()) {
		var lookup models.HarvestDetails
		err := historicCursor.Decode(&lookup)
		if err != nil {
			return nil, fmt.Errorf("error decoding historic harvest: %v", err)
		}
		resultHarvest = append(resultHarvest, lookup)
	}

	if err := historicCursor.Err(); err != nil {
		return nil, err
	}

	return resultHarvest, nil
}

