from determined.experimental import Determined

experiment_id = 701

checkpoint = (
    Determined(master="35.247.36.70")
    .get_experiment(experiment_id)
    .top_checkpoint(sort_by="val_accuracy", smaller_is_better=False)
)

model = checkpoint.load()


print(model.get_weights())