package creating_product

import (
	"context"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/mehdihadeli/go-mediatr"
	"github.com/meysamhadeli/shop-golang-microservices/internal/pkg/logger"
	"github.com/meysamhadeli/shop-golang-microservices/internal/pkg/rabbitmq"
	"github.com/meysamhadeli/shop-golang-microservices/internal/services/product_service/config"
	"github.com/meysamhadeli/shop-golang-microservices/internal/services/product_service/product/consumers"
	creatingproductcommandsv1 "github.com/meysamhadeli/shop-golang-microservices/internal/services/product_service/product/features/creating_product/v1/commands"
	creatingproductdtosv1 "github.com/meysamhadeli/shop-golang-microservices/internal/services/product_service/product/features/creating_product/v1/dtos"
	creatingproducteventsv1 "github.com/meysamhadeli/shop-golang-microservices/internal/services/product_service/product/features/creating_product/v1/events"
	"github.com/meysamhadeli/shop-golang-microservices/internal/services/product_service/shared/delivery"
	test_fixture "github.com/meysamhadeli/shop-golang-microservices/internal/services/product_service/shared/test_fixture"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"testing"
)

type createProductEndToEndTests struct {
	*test_fixture.IntegrationTestFixture
}

var consumer *rabbitmq.Consumer[*delivery.ProductDeliveryBase]

func TestCreateProductEndToEndTest(t *testing.T) {
	//suite.Run(t, &createProductEndToEndTests{IntegrationTestFixture: test_fixture.NewIntegrationTestFixture(t, fx.Options())})
	suite.Run(t, &createProductEndToEndTests{IntegrationTestFixture: test_fixture.NewIntegrationTestFixture(t, fx.Options(
		fx.Invoke(func(ctx context.Context, jaegerTracer trace.Tracer, log logger.ILogger, connRabbitmq *amqp.Connection, cfg *config.Config) {
			consumer = rabbitmq.NewConsumer(cfg.Rabbitmq, connRabbitmq, log, jaegerTracer, consumers.HandleConsumeCreateProduct)
			err := consumer.ConsumeMessage(ctx, creatingproducteventsv1.ProductCreated{}, nil)
			if err != nil {
				require.FailNow(t, err.Error())
			}
		}),
	))})
}

func (c *createProductEndToEndTests) Test_Should_Return_Ok_Status_When_Create_New_Product_To_DB() {

	//tsrv := httptest.NewServer(c.Echo)
	//defer tsrv.Close()
	//
	//e := httpexpect.Default(c.T(), tsrv.URL)
	//
	//request := &v1_dtos.CreateProductRequestDto{
	//	Name:        gofakeit.Name(),
	//	Description: gofakeit.AdjectiveDescriptive(),
	//	Price:       gofakeit.Price(150, 6000),
	//	InventoryId: 1,
	//	Count:       1,
	//}
	//
	//e.POST("/api/v1/products").
	//	WithContext(c.Ctx).
	//	WithJSON(request).
	//	Expect().
	//	Status(http.StatusCreated)
	command := creatingproductcommandsv1.NewCreateProduct(gofakeit.Name(), gofakeit.AdjectiveDescriptive(), gofakeit.Price(150, 6000), 1, 1)
	result, err := mediatr.Send[*creatingproductcommandsv1.CreateProduct, *creatingproductdtosv1.CreateProductResponseDto](c.Ctx, command)
	c.Require().NoError(err)

	isPublished := c.RabbitmqPublisher.IsPublished(creatingproducteventsv1.ProductCreated{})
	c.Assert().Equal(true, isPublished)

	isConsumed := consumer.IsConsumed(creatingproducteventsv1.ProductCreated{})
	c.Assert().Equal(true, isConsumed)

	c.Require().NoError(err)

	c.Assert().NotNil(result)
	c.Assert().Equal(command.ProductID, result.ProductId)

	createdProduct, err := c.IntegrationTestFixture.ProductRepository.GetProductById(c.Ctx, result.ProductId)
	c.Require().NoError(err)
	c.Assert().NotNil(createdProduct)
}

func (c *createProductEndToEndTests) BeforeTest(suiteName, testName string) {
	// some functionality before run tests
}

func (c *createProductEndToEndTests) SetupTest() {
	c.T().Log("SetupTest")
}

func (c *createProductEndToEndTests) TearDownTest() {
	c.T().Log("TearDownTest")
	// cleanup test containers with their hooks
	defer c.PostgresContainer.Terminate(c.Ctx)
	defer c.RabbitmqContainer.Terminate(c.Ctx)
}
