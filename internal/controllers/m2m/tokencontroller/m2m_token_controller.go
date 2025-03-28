// TODO: Describe package
package tokencontroller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/NorskHelsenett/ror-api/internal/apiconnections"
	"github.com/NorskHelsenett/ror-api/internal/models/m2mmodels"
	"github.com/NorskHelsenett/ror-api/internal/models/vaultmodels"

	"github.com/NorskHelsenett/ror/pkg/helpers/rorerror"

	"github.com/NorskHelsenett/ror/pkg/rlog"

	"github.com/NorskHelsenett/ror/pkg/helpers/stringhelper"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
)

func init() {
	rlog.Debug("init m2m token controller")
	validate = validator.New()
}

// TODO: Describe function
//
// TODO: Add swagger
func SelfRegister() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenModel m2mmodels.TokenModel
		ctx := c.Request.Context()
		//validate the request body
		if err := c.BindJSON(&tokenModel); err != nil {
			rerr := rorerror.NewRorError(http.StatusBadRequest, "Missing body", err)
			rerr.GinLogErrorAbort(c)
			return
		}

		//use the validator library to validate required fields
		if err := validate.Struct(&tokenModel); err != nil {
			rerr := rorerror.NewRorError(http.StatusBadRequest, "could not validate input", err)
			rerr.GinLogErrorAbort(c)
			return
		}

		secretPath := fmt.Sprintf("secret/data/v1.0/ror/clusters/%s", tokenModel.ClusterId)
		secret, err := apiconnections.VaultClient.GetSecret(secretPath)
		if err != nil {
			rerr := rorerror.NewRorError(http.StatusBadRequest, "Error checking token", err)
			rerr.GinLogErrorAbort(c, rorerror.Field{Key: "clusterId", Value: tokenModel.ClusterId}, rorerror.Field{Key: "path", Value: secretPath})
			return
		}

		byteArray, err := json.Marshal(secret)
		if err != nil {
			rlog.Errorc(ctx, "could not marshal", err)
		}

		var clusterSecret vaultmodels.VaultClusterModel
		err = json.Unmarshal(byteArray, &clusterSecret)
		if err != nil {
			rlog.Errorc(ctx, "could not unmarshal", err)
		}

		if len(clusterSecret.Data.RorClientSecret) > 0 {
			c.JSON(http.StatusForbidden, nil)
			return
		} else {
			newSecret := stringhelper.RandomString(20, stringhelper.StringTypeAlphaNum)
			clusterSecret.Data.RorClientSecret = newSecret
			secretByteArray, err := json.Marshal(clusterSecret)
			if err != nil {
				rerr := rorerror.NewRorError(http.StatusBadRequest, "A error occured", err)
				rerr.GinLogErrorAbort(c, rorerror.Field{Key: "clusterId", Value: tokenModel.ClusterId})
				return
			}

			_, err = apiconnections.VaultClient.SetSecret(secretPath, secretByteArray)
			if err != nil {
				rerr := rorerror.NewRorError(http.StatusInternalServerError, "A error occured", err)
				rerr.GinLogErrorAbort(c, rorerror.Field{Key: "clusterId", Value: tokenModel.ClusterId})
				return
			}
			tokenModel.Token = clusterSecret.Data.RorClientSecret

			c.JSON(http.StatusOK, tokenModel)
			return
		}
	}
}
